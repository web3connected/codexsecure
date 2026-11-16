package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/web3connected/codexsecure-api/internal/database"
	"github.com/web3connected/codexsecure-api/internal/middleware"
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
)

// AlignmentHandler records Harmonic Alignment / Phase Lock observations of site
// communication relationships (CodexSecure Step 3B; CIR-2026-11-14, HA-2026-11-14).
//
// Authentication reuses the channel routes unchanged: GatewayAssertion verifies the
// gateway's signed assertion for this method, path and {site}, and ChannelAuth requires
// the service key (constant time, closed when unset) and that verified application grant.
// Neither identifies a visitor.
//
// A valid Z1 GET observation is verified against the authoritative PageMeta package
// (method, zone, route, optional developer intent, exact CodexHash v2 evidence) and
// recorded as aligned in harmonic_zones: the relationship row is created on the first
// observation and its alignment_count incremented afterwards. Weights, misalignment and
// Phase Lock scoring are not part of this step. Unknown fields are refused, so no
// identity, session, address, user agent or fingerprint can be carried.
type AlignmentHandler struct {
	store alignmentStore
}

type alignmentStore interface {
	RecordAlignment(ctx context.Context, obs database.AlignmentObservation) (database.AlignmentRecord, error)
}

// NewAlignmentHandler returns the recorder over store (the codex_zones database).
func NewAlignmentHandler(store alignmentStore) *AlignmentHandler {
	return &AlignmentHandler{store: store}
}

// RegisterRoutes adds POST /api/v1/sites/{site}/alignment/relationships/record.
func (h *AlignmentHandler) RegisterRoutes(v1 *gin.RouterGroup) {
	group := v1.Group("/sites/:site/alignment", middleware.GatewayAssertion(), middleware.ChannelAuth())
	group.POST("/relationships/record", h.Record)
}

// alignmentRecordMaxBytes bounds the request body; the contract is a few hundred bytes.
const alignmentRecordMaxBytes = 8 * 1024

var (
	alignmentUUID     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	alignmentZone     = regexp.MustCompile(`^Z([0-9]|1[0-2])$`)
	alignmentNilUUID  = "00000000-0000-0000-0000-000000000000"
	alignmentMaxField = 255
)

const (
	// alignmentRecordedZone is the only zone Step 3B records (S1-1: public Z1 GET).
	alignmentRecordedZone = "Z1"
	// alignmentMaxTIU keeps the TIU within harmonic_zones.last_tiu numeric(20,6).
	alignmentMaxTIU = 1e14
)

// AlignmentRecordRequest is the Z1 GET relationship contract (S1-1): the PageMeta package
// UUID identifies the relationship; zone, route and method describe it; the CodexHash v2
// evidence and CodexTime TIU are the request's existing PageMeta evidence.
type AlignmentRecordRequest struct {
	Relationship    *AlignmentRelationship    `json:"relationship"`
	Evidence        *AlignmentEvidence        `json:"evidence"`
	DeveloperIntent *AlignmentDeveloperIntent `json:"developer_intent,omitempty"`
}

// AlignmentRelationship identifies one communication relationship.
type AlignmentRelationship struct {
	PageMetaUUID string  `json:"pagemeta_uuid"`
	Zone         string  `json:"zone"`
	RouteName    *string `json:"route_name"`
	HTTPMethod   string  `json:"http_method"`
}

// AlignmentEvidence is the proprietary CodexHash v2 and CodexTime evidence.
type AlignmentEvidence struct {
	CodexHash string   `json:"codexhash"`
	TIU       *float64 `json:"tiu"`
}

// AlignmentDeveloperIntent is optional verification context from Core RelationshipContext.
type AlignmentDeveloperIntent struct {
	Intent     *string `json:"intent"`
	Controller *string `json:"controller"`
	Action     *string `json:"action"`
}

// Record validates the observation, verifies it against PageMeta and records it as aligned.
// POST /api/v1/sites/{site}/alignment/relationships/record
func (h *AlignmentHandler) Record(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, alignmentRecordMaxBytes+1))
	if err != nil || len(body) > alignmentRecordMaxBytes {
		alignmentError(c, http.StatusBadRequest, "ALIGNMENT_REQUEST_INVALID", "request body is missing or too large")
		return
	}
	var req AlignmentRecordRequest
	if err := decodeAlignmentRequest(body, &req); err != nil {
		alignmentError(c, http.StatusBadRequest, "ALIGNMENT_REQUEST_INVALID", "request must be one JSON object with only relationship, evidence and developer_intent")
		return
	}
	if field := req.invalidField(); field != "" {
		alignmentError(c, http.StatusUnprocessableEntity, "ALIGNMENT_RELATIONSHIP_INVALID", field+" is invalid")
		return
	}
	if h.store == nil {
		alignmentError(c, http.StatusServiceUnavailable, "ALIGNMENT_STORE_UNAVAILABLE", "alignment store is unavailable")
		return
	}

	obs := database.AlignmentObservation{
		PageMetaUUID: req.Relationship.PageMetaUUID,
		Zone:         req.Relationship.Zone,
		RouteName:    req.Relationship.RouteName,
		HTTPMethod:   req.Relationship.HTTPMethod,
		CodexHash:    req.Evidence.CodexHash,
		TIU:          *req.Evidence.TIU,
	}
	if d := req.DeveloperIntent; d != nil {
		obs.Intent, obs.Controller, obs.Action = d.Intent, d.Controller, d.Action
	}
	record, err := h.store.RecordAlignment(c.Request.Context(), obs)
	var mismatch *database.AlignmentMismatch
	switch {
	case err == nil:
	case errors.As(err, &mismatch):
		alignmentError(c, http.StatusConflict, "ALIGNMENT_RELATIONSHIP_MISMATCH", mismatch.Error())
		return
	case errors.Is(err, database.ErrAlignmentPageMetaNotFound):
		alignmentError(c, http.StatusNotFound, "ALIGNMENT_PAGEMETA_NOT_FOUND", "relationship.pagemeta_uuid is not a recorded pagemeta package")
		return
	case errors.Is(err, database.ErrAlignmentEvidenceMismatch):
		alignmentError(c, http.StatusConflict, "ALIGNMENT_EVIDENCE_MISMATCH", "evidence.codexhash does not match the pagemeta package")
		return
	case errors.Is(err, database.ErrAlignmentRelationshipConflict):
		alignmentError(c, http.StatusConflict, "ALIGNMENT_RELATIONSHIP_CONFLICT", "the stored relationship disagrees with the pagemeta package")
		return
	default:
		// The cause is logged without the request; the client sees a stable code.
		log.Error().Err(err).Str("site", c.Param("site")).Msg("alignment recording failed")
		alignmentError(c, http.StatusServiceUnavailable, "ALIGNMENT_STORE_UNAVAILABLE", "alignment store is unavailable")
		return
	}

	status := http.StatusOK
	if record.Created {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{
		"data": gin.H{
			"recorded": true,
			"result":   record.Result,
			"created":  record.Created,
			"relationship": gin.H{
				"pagemeta_uuid":      record.PageMetaUUID,
				"zone":               record.Zone,
				"route_name":         record.RouteName,
				"http_method":        record.HTTPMethod,
				"alignment_count":    record.AlignmentCount,
				"misalignment_count": record.MisalignmentCount,
			},
		},
	})
}

// decodeAlignmentRequest accepts exactly one JSON object with known fields only.
func decodeAlignmentRequest(body []byte, req *AlignmentRecordRequest) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(req); err != nil {
		return err
	}
	var rest json.RawMessage
	if err := decoder.Decode(&rest); err != io.EOF {
		return errors.New("trailing data")
	}
	return nil
}

// invalidField names the first field that breaks the contract, or "".
func (r AlignmentRecordRequest) invalidField() string {
	rel, ev := r.Relationship, r.Evidence
	switch {
	case rel == nil:
		return "relationship"
	case !alignmentUUID.MatchString(rel.PageMetaUUID) || rel.PageMetaUUID == alignmentNilUUID:
		return "relationship.pagemeta_uuid"
	case !alignmentZone.MatchString(rel.Zone) || rel.Zone != alignmentRecordedZone:
		// Step 3B records the public Z1 GET relationship only.
		return "relationship.zone"
	case rel.RouteName != nil && (*rel.RouteName == "" || len(*rel.RouteName) > alignmentMaxField):
		return "relationship.route_name"
	case rel.HTTPMethod != "GET":
		// The Z1 GET relationship only (S1-1).
		return "relationship.http_method"
	case ev == nil:
		return "evidence"
	case !codexhash.IsV2Evidence(ev.CodexHash):
		return "evidence.codexhash"
	case ev.TIU == nil || math.IsNaN(*ev.TIU) || math.IsInf(*ev.TIU, 0) || *ev.TIU <= 0 || *ev.TIU >= alignmentMaxTIU:
		return "evidence.tiu"
	}
	if d := r.DeveloperIntent; d != nil {
		for name, value := range map[string]*string{"intent": d.Intent, "controller": d.Controller, "action": d.Action} {
			if value != nil && len(*value) > alignmentMaxField {
				return "developer_intent." + name
			}
		}
	}
	return ""
}

func alignmentError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message, "code": code})
}
