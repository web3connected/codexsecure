package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/web3connected/codexsecure-api/internal/database"
	"github.com/web3connected/codexsecure-api/internal/policy"
	"github.com/web3connected/codexsecure-api/internal/protect"
)

// numeric(20,6): up to 14 integer digits and 6 decimals.
var tiuPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,13})(\.[0-9]{1,6})?$`)

// pageMetaStore is the CodexZone PageMeta store (database.DB).
type pageMetaStore interface {
	RecordPageMeta(ctx context.Context, record database.PageMetaRecord) (database.PageMetaResult, error)
}

// PageMetaHandler receives protected PageMeta packages (Stage 5 Record, GET
// PageMeta only) and recognizes or records them in codex_zones. It performs no
// approval, harmonization or enforcement.
type PageMetaHandler struct {
	hasher payloadHasher
	store  pageMetaStore
}

// NewPageMetaHandler creates a PageMeta Record handler.
func NewPageMetaHandler(hasher payloadHasher, store pageMetaStore) *PageMetaHandler {
	return &PageMetaHandler{hasher: hasher, store: store}
}

type pageMetaPackage struct {
	PackageUUID   *string         `json:"package_uuid"`
	Data          json.RawMessage `json:"data"`
	Validation    json.RawMessage `json:"validation"`
	SubmittedJSON *string         `json:"submitted_json"`
	Security      *struct {
		CodexHash string      `json:"codexhash"`
		TIU       json.Number `json:"tiu"`
	} `json:"security"`
}

type pageMetaIdentity struct {
	Method string `json:"method"`
	Route  *struct {
		URI  string  `json:"uri"`
		Name *string `json:"name"`
	} `json:"route"`
	DeclaredZone string `json:"declared_zone"`
}

// Record recognizes or records a protected PageMeta package.
// POST /api/v1/pagemeta/record
func (h *PageMetaHandler) Record(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, protectMaxBodyBytes)

	var req struct {
		Package json.RawMessage `json:"package"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			pageMetaError(c, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body exceeds 1 MiB")
			return
		}
		pageMetaError(c, http.StatusBadRequest, "invalid_request", "Request body must be a JSON object with a package field")
		return
	}
	raw := bytes.TrimSpace(req.Package)
	if len(raw) == 0 || raw[0] != '{' {
		pageMetaError(c, http.StatusBadRequest, "invalid_request", "package must be a JSON object")
		return
	}

	var pkg pageMetaPackage
	if err := json.Unmarshal(raw, &pkg); err != nil {
		pageMetaError(c, http.StatusUnprocessableEntity, "invalid_package", "package fields have invalid types")
		return
	}

	canonical, identity, code, message := validatePageMetaPackage(pkg)
	if code != "" {
		pageMetaError(c, http.StatusUnprocessableEntity, code, message)
		return
	}

	// The submitted CodexHash must belong to this data: recompute it through
	// CodexHash (stateless SHA-512 of the canonical data). No local hashing.
	computed, err := h.hasher.PayloadHash(c.Request.Context(), canonical, protectHashAlgo, protectHashSize)
	if err != nil || computed.Algo != protectHashAlgo || !sha512Hex.MatchString(computed.PayloadHash) {
		log.Warn().Str("operation", "pagemeta.record").Msg("CodexHash unavailable for PageMeta verification")
		pageMetaError(c, http.StatusServiceUnavailable, "codexhash_unavailable", "CodexHash could not verify the package hash")
		return
	}
	if computed.PayloadHash != pkg.Security.CodexHash {
		pageMetaError(c, http.StatusUnprocessableEntity, "hash_mismatch", "security.codexhash does not match the package data")
		return
	}

	result, err := h.store.RecordPageMeta(c.Request.Context(), database.PageMetaRecord{
		Method:  identity.Method,
		URI:     identity.Route.URI,
		Hash:    pkg.Security.CodexHash,
		TIU:     pkg.Security.TIU.String(),
		Payload: raw,
	})
	if err != nil {
		log.Warn().Str("operation", "pagemeta.record").Msg("CodexZone PageMeta recording failed")
		pageMetaError(c, http.StatusServiceUnavailable, "database_unavailable", "The PageMeta package could not be recorded")
		return
	}

	page := gin.H{"method": identity.Method, "uri": identity.Route.URI, "route_name": identity.Route.Name}
	log.Info().Str("operation", "pagemeta.record").Str("result", result.Result).
		Str("method", identity.Method).Str("uri", identity.Route.URI).Msg("PageMeta package processed")

	status := http.StatusOK
	if result.Recorded {
		status = http.StatusCreated
	}
	body := gin.H{
		"result":         result.Result,
		"recorded":       result.Recorded,
		"package":        result.Package,
		"known_packages": result.KnownPackages,
		"page":           page,
	}
	if result.Result == database.PageMetaConfigurationChanged {
		body["message"] = "The page is already recorded with a different structural configuration. The changed package was not recorded: no candidate or approval state exists."
	}
	c.JSON(status, body)
}

// validatePageMetaPackage checks structure and returns the canonical data form
// and page identity, or an error code and message.
func validatePageMetaPackage(pkg pageMetaPackage) (string, pageMetaIdentity, string, string) {
	var identity pageMetaIdentity

	if pkg.PackageUUID != nil {
		if _, err := uuid.Parse(*pkg.PackageUUID); err != nil {
			return "", identity, "invalid_package", "package_uuid must be null or a UUID"
		}
	}

	canonical, err := protect.CanonicalPayload(pkg.Data)
	switch {
	case errors.Is(err, protect.ErrReservedField):
		return "", identity, "reserved_field", "data must not contain codexhash or tiu"
	case errors.Is(err, protect.ErrDuplicateKey):
		return "", identity, "duplicate_key", "data contains a duplicate object key"
	case err != nil:
		return "", identity, "invalid_package", "data must be a non-empty JSON object"
	}

	if err := json.Unmarshal(pkg.Data, &identity); err != nil {
		return "", identity, "invalid_package", "data.method, data.route.uri and data.declared_zone must be strings"
	}
	if !policy.ValidMethods[identity.Method] {
		return "", identity, "invalid_package", "data.method must be an uppercase HTTP method"
	}
	if identity.Route == nil || strings.TrimSpace(identity.Route.URI) == "" {
		return "", identity, "invalid_package", "data.route.uri is required"
	}
	if _, ok := policy.ZoneCodeToID(identity.DeclaredZone); !ok {
		return "", identity, "invalid_package", "data.declared_zone must be a zone code Z0-Z12"
	}

	validation := bytes.TrimSpace(pkg.Validation)
	if len(validation) > 0 && !bytes.Equal(validation, []byte("null")) && validation[0] != '{' {
		return "", identity, "invalid_package", "validation must be an object or null"
	}

	if pkg.SubmittedJSON != nil {
		var submitted struct {
			Payload json.RawMessage `json:"payload"`
		}
		if json.Unmarshal([]byte(*pkg.SubmittedJSON), &submitted) != nil {
			return "", identity, "submitted_json_mismatch", "submitted_json must be the {\"payload\": data} transport form"
		}
		if submittedCanonical, err := protect.CanonicalPayload(submitted.Payload); err != nil || submittedCanonical != canonical {
			return "", identity, "submitted_json_mismatch", "submitted_json does not carry the package data"
		}
	}

	if pkg.Security == nil || !sha512Hex.MatchString(pkg.Security.CodexHash) {
		return "", identity, "invalid_package", "security.codexhash must be a lowercase SHA-512 hex string"
	}
	tiu := pkg.Security.TIU.String()
	if !tiuPattern.MatchString(tiu) || strings.Trim(strings.ReplaceAll(tiu, ".", ""), "0") == "" {
		return "", identity, "invalid_package", "security.tiu must be a positive number with at most 6 decimals"
	}

	return canonical, identity, "", ""
}

func pageMetaError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": code, "message": message})
}
