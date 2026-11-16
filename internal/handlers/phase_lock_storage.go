package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/web3connected/codexsecure-api/internal/database"
	"github.com/web3connected/codexsecure-api/internal/middleware"
	"github.com/web3connected/codexsecure-api/internal/protect"
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
)

// PhaseLockStorageHandler serves persistent Phase Lock structure and pattern storage
// (CodexSecure P3; PL-2026-11-15, PLA-2026-11-15; schema P2) for the Laravel Core
// FormStructureStore contract: record, patterns, pattern, discover.
//
// Authentication is the alignment and channel contract, unchanged: GatewayAssertion
// verifies the gateway's signed assertion for this method, path and {site}, and
// ChannelAuth requires the service key (constant time, closed when unset) and that
// verified application grant. {site} authorizes the call only: the P2 tables are
// site-less (I19, PL-15), so no per-site isolation of the stored patterns is claimed.
//
// A recorded structure is never trusted because the caller computed it: its CodexHash v2
// evidence is recomputed here through the proprietary engine over the canonical document
// (codexsecure-protect-payload-v1, as Protect and PageMeta), with no local or
// conventional fallback. Documents are values-free structures (method, destination,
// fields); unknown members, numbers, values and payloads are refused. Nothing here
// approves, rejects, writes a decision or event, or reinforces a pattern; a caller-
// supplied state is never accepted (there is no field for it).
type PhaseLockStorageHandler struct {
	v2    v2Hasher
	store phaseLockStore
}

type phaseLockStore interface {
	RecordFormStructure(ctx context.Context, record database.FormStructureRecord) (string, database.FormStructurePackage, error)
	RelationshipPatterns(ctx context.Context, rel database.PhaseLockRelationship) ([]database.PhaseLockEntry, error)
	Pattern(ctx context.Context, rel database.PhaseLockRelationship, structureCodexHash string) (*database.PhaseLockPattern, error)
	DiscoverPattern(ctx context.Context, rel database.PhaseLockRelationship, structureCodexHash, authority string) (string, *database.PhaseLockPattern, error)
}

// NewPhaseLockStorageHandler returns the storage API over the CodexHash v2 engine and
// the codex_zones database.
func NewPhaseLockStorageHandler(hasher v2Hasher, store phaseLockStore) *PhaseLockStorageHandler {
	return &PhaseLockStorageHandler{v2: hasher, store: store}
}

// RegisterRoutes adds the four POST operations under /api/v1/sites/{site}/phase-lock.
// The legacy /api/v1/phase-lock/record score endpoint is a different route and unchanged.
func (h *PhaseLockStorageHandler) RegisterRoutes(v1 *gin.RouterGroup) {
	group := v1.Group("/sites/:site/phase-lock", middleware.GatewayAssertion(), middleware.ChannelAuth())
	group.POST("/structures/record", h.RecordStructure)
	group.POST("/relationships/patterns", h.RelationshipPatterns)
	group.POST("/patterns/lookup", h.Pattern)
	group.POST("/patterns/discover", h.Discover)
}

// phaseLockMaxBytes bounds a request; a form structure is a few kilobytes.
const phaseLockMaxBytes = 64 * 1024

var (
	phaseLockZone      = regexp.MustCompile(`^Z([0-9]|1[0-2])$`)
	phaseLockMaxRoute  = 255
	phaseLockMethods   = map[string]bool{"POST": true} // FormStructure::METHODS
	phaseLockParts     = map[string]bool{"form": true, "files": true}
	phaseLockPresences = map[string]bool{"always": true, "conditional": true}
)

// phaseLockRelationshipBody is the relationship as FormStructure::relationship() names it.
type phaseLockRelationshipBody struct {
	Zone   *string `json:"zone"`
	Route  *string `json:"route"`
	Method *string `json:"method"`
}

type phaseLockStructureRecordRequest struct {
	Relationship  *phaseLockRelationshipBody `json:"relationship"`
	Data          json.RawMessage            `json:"data"`
	SubmittedJSON *string                    `json:"submitted_json"`
	Security      *struct {
		CodexHash *string         `json:"codexhash"`
		TIU       json.RawMessage `json:"tiu"`
	} `json:"security"`
}

type phaseLockPatternsRequest struct {
	Relationship *phaseLockRelationshipBody `json:"relationship"`
}

type phaseLockLookupRequest struct {
	Relationship       *phaseLockRelationshipBody `json:"relationship"`
	StructureCodexHash *string                    `json:"structure_codexhash"`
}

type phaseLockDiscoverRequest struct {
	Relationship       *phaseLockRelationshipBody `json:"relationship"`
	StructureCodexHash *string                    `json:"structure_codexhash"`
	Authority          *string                    `json:"authority"`
}

// formStructureDocument mirrors codex_form_structure_valid(jsonb) (P2) and
// FormStructure::document(). Pointers distinguish a missing or null member.
type formStructureDocument struct {
	Method      *string `json:"method"`
	Destination *struct {
		Zone       *string         `json:"zone"`
		Route      json.RawMessage `json:"route"`
		URI        *string         `json:"uri"`
		Controller json.RawMessage `json:"controller"`
		Action     json.RawMessage `json:"action"`
	} `json:"destination"`
	Fields *[]*struct {
		Name     *string    `json:"name"`
		Part     *string    `json:"part"`
		Controls *[]*string `json:"controls"`
		Presence *string    `json:"presence"`
		Required *bool      `json:"required"`
	} `json:"fields"`
}

// RecordStructure verifies and records one captured form structure.
// POST /api/v1/sites/{site}/phase-lock/structures/record
func (h *PhaseLockStorageHandler) RecordStructure(c *gin.Context) {
	var req phaseLockStructureRecordRequest
	if !decodePhaseLockRequest(c, &req) {
		return
	}
	rel, field := phaseLockRelationship(req.Relationship)
	if field != "" {
		phaseLockFieldError(c, field)
		return
	}
	canonical, routeOrURI, field := validateFormStructure(req.Data)
	if field != "" {
		phaseLockError(c, http.StatusUnprocessableEntity, "PHASE_LOCK_STRUCTURE_INVALID", field+" is invalid")
		return
	}
	if req.SubmittedJSON != nil {
		var submitted struct {
			Payload json.RawMessage `json:"payload"`
		}
		if json.Unmarshal([]byte(*req.SubmittedJSON), &submitted) != nil {
			phaseLockFieldError(c, "submitted_json")
			return
		}
		if again, err := protect.CanonicalPayload(submitted.Payload); err != nil || again != canonical {
			phaseLockError(c, http.StatusUnprocessableEntity, "PHASE_LOCK_STRUCTURE_INVALID", "submitted_json does not carry the structure data")
			return
		}
	}
	sec := req.Security
	switch {
	case sec == nil:
		phaseLockFieldError(c, "security")
		return
	case sec.CodexHash == nil || !codexhash.IsV2Evidence(*sec.CodexHash):
		phaseLockFieldError(c, "security.codexhash")
		return
	case !phaseLockTIU(sec.TIU):
		phaseLockFieldError(c, "security.tiu")
		return
	}

	// The document must describe the relationship it is recorded under.
	var doc struct {
		Method      string `json:"method"`
		Destination struct {
			Zone string `json:"zone"`
		} `json:"destination"`
	}
	_ = json.Unmarshal([]byte(canonical), &doc)
	switch {
	case doc.Method != rel.Method:
		phaseLockMismatch(c, "data.method")
		return
	case doc.Destination.Zone != rel.Zone:
		phaseLockMismatch(c, "data.destination.zone")
		return
	case routeOrURI != rel.Route:
		phaseLockMismatch(c, "data.destination.route")
		return
	}

	// Recompute the CodexHash v2 evidence through the proprietary engine; never trust the caller's.
	if h.v2 == nil {
		log.Warn().Str("operation", "phase_lock.record").Msg("CodexHash v2 not configured for structure verification")
		phaseLockError(c, http.StatusServiceUnavailable, "PHASE_LOCK_CODEXHASH_UNAVAILABLE", "CodexHash could not verify the structure evidence")
		return
	}
	computed, err := h.v2.V2Hash(c.Request.Context(), canonical)
	if err != nil {
		log.Warn().Str("operation", "phase_lock.record").Bool("invalid_response", errors.Is(err, codexhash.ErrInvalidResponse)).
			Msg("CodexHash v2 unavailable for structure verification")
		phaseLockError(c, http.StatusServiceUnavailable, "PHASE_LOCK_CODEXHASH_UNAVAILABLE", "CodexHash could not verify the structure evidence")
		return
	}
	if computed.Digest != *sec.CodexHash {
		phaseLockError(c, http.StatusUnprocessableEntity, "PHASE_LOCK_EVIDENCE_MISMATCH", "security.codexhash does not match the structure data")
		return
	}
	if h.store == nil {
		phaseLockUnavailable(c, "phase_lock.record", nil)
		return
	}

	result, pkg, err := h.store.RecordFormStructure(c.Request.Context(), database.FormStructureRecord{
		Relationship: rel, Canonical: canonical, CodexHash: *sec.CodexHash, TIU: string(bytes.TrimSpace(sec.TIU)),
	})
	if err != nil {
		phaseLockStoreError(c, "phase_lock.record", err)
		return
	}
	phaseLockLog(c, "phase_lock.record", result, rel)
	status := http.StatusOK
	if result == database.PhaseLockRecorded {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{"data": gin.H{"result": result, "package": phaseLockPackageJSON(pkg)}})
}

// RelationshipPatterns lists every structure package of one relationship with its pattern.
// POST /api/v1/sites/{site}/phase-lock/relationships/patterns
func (h *PhaseLockStorageHandler) RelationshipPatterns(c *gin.Context) {
	var req phaseLockPatternsRequest
	if !decodePhaseLockRequest(c, &req) {
		return
	}
	rel, field := phaseLockRelationship(req.Relationship)
	if field != "" {
		phaseLockFieldError(c, field)
		return
	}
	if h.store == nil {
		phaseLockUnavailable(c, "phase_lock.patterns", nil)
		return
	}
	entries, err := h.store.RelationshipPatterns(c.Request.Context(), rel)
	if err != nil {
		phaseLockStoreError(c, "phase_lock.patterns", err)
		return
	}
	out := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
		out = append(out, gin.H{"package": phaseLockPackageJSON(entry.Package), "pattern": phaseLockPatternJSON(entry.Pattern)})
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"result": database.PhaseLockFound, "relationship": rel, "entries": out}})
}

// Pattern returns one pattern by its relationship and structure CodexHash.
// POST /api/v1/sites/{site}/phase-lock/patterns/lookup
func (h *PhaseLockStorageHandler) Pattern(c *gin.Context) {
	var req phaseLockLookupRequest
	if !decodePhaseLockRequest(c, &req) {
		return
	}
	rel, field := phaseLockRelationship(req.Relationship)
	if field == "" && (req.StructureCodexHash == nil || !codexhash.IsV2Evidence(*req.StructureCodexHash)) {
		field = "structure_codexhash"
	}
	if field != "" {
		phaseLockFieldError(c, field)
		return
	}
	if h.store == nil {
		phaseLockUnavailable(c, "phase_lock.pattern", nil)
		return
	}
	pattern, err := h.store.Pattern(c.Request.Context(), rel, *req.StructureCodexHash)
	if err != nil {
		phaseLockStoreError(c, "phase_lock.pattern", err)
		return
	}
	result := database.PhaseLockFound
	if pattern == nil {
		result = database.PhaseLockNotFound
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"result": result, "pattern": phaseLockPatternJSON(pattern)}})
}

// Discover turns a recorded structure into a pattern under the requested authority.
// POST /api/v1/sites/{site}/phase-lock/patterns/discover
func (h *PhaseLockStorageHandler) Discover(c *gin.Context) {
	var req phaseLockDiscoverRequest
	if !decodePhaseLockRequest(c, &req) {
		return
	}
	rel, field := phaseLockRelationship(req.Relationship)
	switch {
	case field != "":
	case req.StructureCodexHash == nil || !codexhash.IsV2Evidence(*req.StructureCodexHash):
		field = "structure_codexhash"
	case req.Authority == nil || (*req.Authority != database.PhaseLockDeveloperAdmin && *req.Authority != database.PhaseLockCommunicationObject):
		field = "authority"
	}
	if field != "" {
		phaseLockFieldError(c, field)
		return
	}
	if h.store == nil {
		phaseLockUnavailable(c, "phase_lock.discover", nil)
		return
	}
	result, pattern, err := h.store.DiscoverPattern(c.Request.Context(), rel, *req.StructureCodexHash, *req.Authority)
	if err != nil {
		phaseLockStoreError(c, "phase_lock.discover", err)
		return
	}
	phaseLockLog(c, "phase_lock.discover", result, rel)
	status := http.StatusOK
	if result == database.PhaseLockDiscovered {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{"data": gin.H{"result": result, "pattern": phaseLockPatternJSON(pattern)}})
}

// decodePhaseLockRequest reads one bounded JSON object with known members only.
func decodePhaseLockRequest(c *gin.Context, out any) bool {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, phaseLockMaxBytes+1))
	if err != nil || len(body) > phaseLockMaxBytes {
		phaseLockError(c, http.StatusBadRequest, "PHASE_LOCK_REQUEST_INVALID", "request body is missing or too large")
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(out) != nil {
		phaseLockError(c, http.StatusBadRequest, "PHASE_LOCK_REQUEST_INVALID", "request must be one JSON object with only the documented fields")
		return false
	}
	var rest json.RawMessage
	if decoder.Decode(&rest) != io.EOF {
		phaseLockError(c, http.StatusBadRequest, "PHASE_LOCK_REQUEST_INVALID", "request must be one JSON object with only the documented fields")
		return false
	}
	return true
}

// phaseLockRelationship validates the relationship, or names the first invalid field.
func phaseLockRelationship(body *phaseLockRelationshipBody) (database.PhaseLockRelationship, string) {
	switch {
	case body == nil:
		return database.PhaseLockRelationship{}, "relationship"
	case body.Zone == nil || !phaseLockZone.MatchString(*body.Zone):
		return database.PhaseLockRelationship{}, "relationship.zone"
	case body.Route == nil || strings.TrimSpace(*body.Route) == "" || len(*body.Route) > phaseLockMaxRoute:
		return database.PhaseLockRelationship{}, "relationship.route"
	case body.Method == nil || !phaseLockMethods[*body.Method]:
		return database.PhaseLockRelationship{}, "relationship.method"
	}
	return database.PhaseLockRelationship{Zone: *body.Zone, Route: *body.Route, Method: *body.Method}, ""
}

// validateFormStructure returns the canonical form of a values-free structure document and
// its route-or-URI, or names the first invalid member. It accepts exactly what
// codex_form_structure_valid(jsonb) accepts: strings, booleans and the nullable route,
// controller and action; never a number, value or extra member.
func validateFormStructure(raw json.RawMessage) (string, string, string) {
	canonical, err := protect.CanonicalPayload(raw)
	if err != nil {
		return "", "", "data"
	}
	decoder := json.NewDecoder(strings.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var doc formStructureDocument
	if decoder.Decode(&doc) != nil {
		return "", "", "data"
	}
	if doc.Method == nil {
		return "", "", "data.method"
	}
	d := doc.Destination
	if d == nil {
		return "", "", "data.destination"
	}
	route, routeOK := nullableString(d.Route)
	switch {
	case d.Zone == nil:
		return "", "", "data.destination.zone"
	case d.URI == nil:
		return "", "", "data.destination.uri"
	case !routeOK:
		return "", "", "data.destination.route"
	}
	if _, ok := nullableString(d.Controller); !ok {
		return "", "", "data.destination.controller"
	}
	if _, ok := nullableString(d.Action); !ok {
		return "", "", "data.destination.action"
	}
	if doc.Fields == nil {
		return "", "", "data.fields"
	}
	for _, f := range *doc.Fields {
		switch {
		case f == nil, f.Name == nil, *f.Name == "", f.Part == nil, !phaseLockParts[*f.Part],
			f.Presence == nil, !phaseLockPresences[*f.Presence], f.Required == nil, f.Controls == nil:
			return "", "", "data.fields"
		}
		for _, control := range *f.Controls {
			if control == nil {
				return "", "", "data.fields"
			}
		}
	}
	routeOrURI := *d.URI
	if route != nil {
		routeOrURI = *route
	}
	return canonical, routeOrURI, ""
}

// nullableString accepts a present member that is a string or null.
func nullableString(raw json.RawMessage) (*string, bool) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, false // missing
	}
	if bytes.Equal(trimmed, []byte("null")) {
		return nil, true
	}
	var value string
	if trimmed[0] != '"' || json.Unmarshal(trimmed, &value) != nil {
		return nil, false
	}
	return &value, true
}

// phaseLockTIU accepts a positive JSON number within numeric(20,6), as PageMeta does.
func phaseLockTIU(raw json.RawMessage) bool {
	tiu := string(bytes.TrimSpace(raw))
	return tiuPattern.MatchString(tiu) && strings.Trim(strings.ReplaceAll(tiu, ".", ""), "0") != ""
}

func phaseLockPackageJSON(pkg database.FormStructurePackage) gin.H {
	return gin.H{
		"relationship": pkg.Relationship,
		"data":         pkg.Data,
		"security":     gin.H{"codexhash": pkg.CodexHash, "tiu": json.RawMessage(pkg.TIU)},
		"created_at":   pkg.CreatedAt,
	}
}

// phaseLockPatternJSON is the Core pattern shape; state is null under Communications.
func phaseLockPatternJSON(pattern *database.PhaseLockPattern) any {
	if pattern == nil {
		return nil
	}
	return gin.H{
		"relationship":        pattern.Relationship,
		"structure_codexhash": pattern.StructureCodexHash,
		"authority":           pattern.Authority,
		"state":               pattern.State,
	}
}

func phaseLockStoreError(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, database.ErrPhaseLockStructureRejected):
		log.Warn().Err(err).Str("operation", operation).Str("site", c.Param("site")).Msg("phase lock schema refused the structure")
		phaseLockError(c, http.StatusUnprocessableEntity, "PHASE_LOCK_STRUCTURE_INVALID", "the structure is not a valid values-free structure of the relationship")
	case errors.Is(err, database.ErrPhaseLockStructureConflict):
		log.Error().Str("operation", operation).Str("site", c.Param("site")).Msg("stored phase lock structure does not reproduce its evidence")
		phaseLockError(c, http.StatusConflict, "PHASE_LOCK_STRUCTURE_CONFLICT", "the stored structure for this evidence does not match")
	default:
		phaseLockUnavailable(c, operation, err)
	}
}

func phaseLockUnavailable(c *gin.Context, operation string, err error) {
	// The cause is logged without the request; the client sees a stable code.
	log.Error().Err(err).Str("operation", operation).Str("site", c.Param("site")).Msg("phase lock store unavailable")
	phaseLockError(c, http.StatusServiceUnavailable, "PHASE_LOCK_STORE_UNAVAILABLE", "phase lock store is unavailable")
}

func phaseLockLog(c *gin.Context, operation, result string, rel database.PhaseLockRelationship) {
	log.Info().Str("operation", operation).Str("site", c.Param("site")).Str("result", result).
		Str("zone", rel.Zone).Str("route", rel.Route).Str("method", rel.Method).Msg("phase lock storage")
}

func phaseLockFieldError(c *gin.Context, field string) {
	phaseLockError(c, http.StatusUnprocessableEntity, "PHASE_LOCK_FIELD_INVALID", field+" is invalid")
}

func phaseLockMismatch(c *gin.Context, field string) {
	phaseLockError(c, http.StatusUnprocessableEntity, "PHASE_LOCK_RELATIONSHIP_MISMATCH", field+" does not match the relationship")
}

func phaseLockError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message, "code": code})
}
