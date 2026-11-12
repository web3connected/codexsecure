package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/web3connected/codexsecure-api/internal/communications"
	"github.com/web3connected/codexsecure-api/internal/middleware"
)

type CommunicationHandler struct{ store communications.Store }

func NewCommunicationHandler(store communications.Store) *CommunicationHandler {
	return &CommunicationHandler{store: store}
}

// RegisterRoutes uses the existing API prefix. Gateway adds /secure externally.
// None of these routes participates in browser/public zone registration.
func (h *CommunicationHandler) RegisterRoutes(v1 *gin.RouterGroup) {
	group := v1.Group("/sites/:site/communications", middleware.GatewayAssertion(), middleware.CommunicationAuth())
	group.GET("/assessments", h.Assess)
	group.GET("/snapshot", h.Snapshot)
	group.POST("/discovery", h.Discover)
	group.POST("/discovery/batches", h.DiscoveryBatch("start"))
	group.POST("/discovery/batches/:batch/failures", h.DiscoveryBatch("failure"))
	group.POST("/discovery/batches/:batch/finalize", h.DiscoveryBatch("finalize"))
	group.POST("/:key/register", h.Register)
	group.GET("/:key/configuration", h.Find)
	group.GET("/:key/configuration/history", h.History)
	group.PUT("/:key/configuration/draft", h.Decide("draft"))
	group.POST("/:key/configuration/approve", h.Decide("approve"))
	group.POST("/:key/configuration/reject", h.Decide("reject"))
	group.POST("/:key/disable", h.Decide("disable"))
	group.POST("/:key/revoke", h.Decide("revoke"))
}
func communicationFailure(c *gin.Context, e error) {
	api := communications.MapError(e)
	c.JSON(api.Status, api)
}
func communicationIdentity(c *gin.Context) bool {
	site := c.Param("site")
	if site == "" || len([]rune(site)) > 128 || strings.TrimSpace(site) != site {
		communicationFailure(c, communications.ValidationErrors{"site": {"invalid site key"}})
		return false
	}
	if key := c.Param("key"); key != "" {
		id, e := uuid.Parse(key)
		if e != nil || id == uuid.Nil {
			communicationFailure(c, communications.ValidationErrors{"registration_key": {"must be a nonzero UUID"}})
			return false
		}
	}
	return true
}
func readCommunicationBody(c *gin.Context, target any) bool {
	raw, e := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10))
	if e != nil {
		communicationFailure(c, communications.APIError{Status: 413, Code: "CONFIGURATION_INVALID", Message: "request body exceeds limit"})
		return false
	}
	if e = communications.DecodeStrict(raw, target); e != nil || string(raw) == "null" {
		communicationFailure(c, communications.APIError{Status: 400, Code: "CONFIGURATION_INVALID", Message: "invalid JSON request", Errors: communications.ValidationErrors{"request": {"must be one JSON object with supported fields"}}})
		return false
	}
	// DecodeStrict enforces field names/types. Check the top-level kind explicitly,
	// including whitespace around null, which otherwise decodes into a nil pointer.
	var object map[string]json.RawMessage
	if e = json.Unmarshal(raw, &object); e != nil || object == nil {
		communicationFailure(c, communications.ValidationErrors{"request": {"must be a JSON object"}})
		return false
	}
	return true
}
func (h *CommunicationHandler) Find(c *gin.Context) {
	if !communicationIdentity(c) {
		return
	}
	result, e := h.store.Find(c.Request.Context(), c.Param("site"), c.Param("key"))
	if e != nil {
		communicationFailure(c, e)
		return
	}
	c.JSON(200, gin.H{"data": result})
}
func (h *CommunicationHandler) Snapshot(c *gin.Context) {
	if !communicationIdentity(c) {
		return
	}
	store, ok := h.store.(communications.SnapshotStore)
	if !ok {
		communicationFailure(c, communications.APIError{Status: 503, Code: "SNAPSHOT_UNAVAILABLE", Message: "snapshot service unavailable"})
		return
	}
	result, e := store.Snapshot(c.Request.Context(), c.Param("site"))
	if e != nil {
		communicationFailure(c, e)
		return
	}
	c.JSON(200, gin.H{"data": result})
}
func (h *CommunicationHandler) History(c *gin.Context) {
	if !communicationIdentity(c) {
		return
	}
	result, e := h.store.History(c.Request.Context(), c.Param("site"), c.Param("key"))
	if e != nil {
		communicationFailure(c, e)
		return
	}
	c.JSON(200, gin.H{"data": result})
}
func (h *CommunicationHandler) Register(c *gin.Context) {
	if !communicationIdentity(c) {
		return
	}
	store, ok := h.store.(communications.RegistrationStore)
	if !ok {
		communicationFailure(c, communications.APIError{Status: 503, Code: "INTERNAL_ERROR", Message: "registration service unavailable"})
		return
	}
	var body communications.RegistrationRequest
	if !readCommunicationBody(c, &body) {
		return
	}
	result, e := store.Register(c.Request.Context(), c.Param("site"), c.Param("key"), middleware.CommunicationActor(c), c.GetHeader("Idempotency-Key"), body)
	if e != nil {
		communicationFailure(c, e)
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	c.JSON(status, gin.H{"data": result})
}
func (h *CommunicationHandler) Decide(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !communicationIdentity(c) {
			return
		}
		d := communications.Decision{Action: action, Site: c.Param("site"), Key: c.Param("key"), Actor: middleware.CommunicationActor(c), IdempotencyKey: c.GetHeader("Idempotency-Key")}
		switch action {
		case "draft":
			var body struct {
				Document        json.RawMessage `json:"document"`
				ExpectedVersion *int64          `json:"expected_version"`
			}
			if !readCommunicationBody(c, &body) {
				return
			}
			d.Document = body.Document
			d.ExpectedVersion = body.ExpectedVersion
		case "approve":
			var body struct {
				DraftVersion      *int64  `json:"draft_version"`
				StructureRevision *int64  `json:"structure_revision"`
				Reason            *string `json:"reason"`
			}
			if !readCommunicationBody(c, &body) {
				return
			}
			d.DraftVersion = body.DraftVersion
			d.StructureRevision = body.StructureRevision
			d.Reason = body.Reason
		case "reject":
			var body struct {
				DraftVersion *int64  `json:"draft_version"`
				Reason       *string `json:"reason"`
			}
			if !readCommunicationBody(c, &body) {
				return
			}
			d.DraftVersion = body.DraftVersion
			d.Reason = body.Reason
		default:
			var body struct {
				Reason *string `json:"reason"`
			}
			if !readCommunicationBody(c, &body) {
				return
			}
			d.Reason = body.Reason
		}
		result, e := h.store.Decide(c.Request.Context(), d)
		if e != nil {
			communicationFailure(c, e)
			return
		}
		c.JSON(200, gin.H{"data": result})
	}
}
func (h *CommunicationHandler) Discover(c *gin.Context) {
	if !communicationIdentity(c) {
		return
	}
	var body communications.Discovery
	if !readCommunicationBody(c, &body) {
		return
	}
	result, e := h.store.Discover(c.Request.Context(), c.Param("site"), c.GetHeader("Idempotency-Key"), body)
	if e != nil {
		communicationFailure(c, e)
		return
	}
	c.JSON(200, gin.H{"data": result})
}
func (h *CommunicationHandler) Assess(c *gin.Context) {
	if !communicationIdentity(c) {
		return
	}
	query := c.Request.URL.Query()
	for key, values := range query {
		if (key != "scope" && key != "page") || len(values) != 1 {
			communicationFailure(c, communications.ValidationErrors{"scope": {"unsupported or repeated query parameter"}})
			return
		}
	}
	result, e := h.store.Assess(c.Request.Context(), c.Param("site"), communications.Coverage{Type: c.Query("scope"), Page: c.Query("page")})
	if e != nil {
		api := communications.MapError(e)
		if api.Status >= 500 {
			result.Complete = false
			c.JSON(api.Status, gin.H{"error": api.Message, "code": api.Code, "data": result})
			return
		}
		communicationFailure(c, e)
		return
	}
	c.JSON(200, gin.H{"data": result})
}

func (h *CommunicationHandler) DiscoveryBatch(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !communicationIdentity(c) {
			return
		}
		store, ok := h.store.(communications.BatchStore)
		if !ok {
			communicationFailure(c, communications.APIError{Status: 503, Code: "INTERNAL_ERROR", Message: "discovery service unavailable"})
			return
		}
		var result json.RawMessage
		var e error
		switch action {
		case "start":
			var body communications.DiscoveryStart
			if !readCommunicationBody(c, &body) {
				return
			}
			result, e = store.StartDiscovery(c.Request.Context(), c.Param("site"), c.GetHeader("Idempotency-Key"), middleware.CommunicationActor(c), body)
		case "failure":
			var body communications.DiscoveryFailure
			if !readCommunicationBody(c, &body) {
				return
			}
			result, e = store.RecordDiscoveryFailure(c.Request.Context(), c.Param("site"), c.Param("batch"), body)
		case "finalize":
			var body communications.DiscoveryFinalize
			if !readCommunicationBody(c, &body) {
				return
			}
			result, e = store.FinalizeDiscovery(c.Request.Context(), c.Param("site"), c.Param("batch"), middleware.CommunicationActor(c), body)
		}
		if e != nil {
			communicationFailure(c, e)
			return
		}
		c.JSON(200, gin.H{"data": result})
	}
}
