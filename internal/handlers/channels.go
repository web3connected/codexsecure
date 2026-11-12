package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/web3connected/codexsecure-api/internal/channels"
	"github.com/web3connected/codexsecure-api/internal/communications"
	"github.com/web3connected/codexsecure-api/internal/middleware"
)

// channelManager is the CommunicationChannelManager surface exposed over HTTP.
// Expiry is manager-internal and deliberately absent.
type channelManager interface {
	Open(context.Context, string, string) (channels.Result, error)
	Get(context.Context, string, uuid.UUID) (channels.Result, error)
	Close(context.Context, string, uuid.UUID) (channels.Result, error)
	Activate(context.Context, string, uuid.UUID) (channels.Result, error)
	Renew(context.Context, string, uuid.UUID) (channels.Result, error)
	BindIdentity(context.Context, string, uuid.UUID, uuid.UUID) (channels.Result, error)
}

type ChannelHandler struct{ manager channelManager }

func NewChannelHandler(manager channelManager) *ChannelHandler {
	return &ChannelHandler{manager: manager}
}

// RegisterRoutes uses the existing API prefix. Gateway adds /secure externally.
// GatewayAssertion installs the verified gateway principal that ChannelAuth requires.
func (h *ChannelHandler) RegisterRoutes(v1 *gin.RouterGroup) {
	group := v1.Group("/sites/:site/channels", middleware.GatewayAssertion(), middleware.ChannelAuth())
	group.POST("", h.Open)
	group.GET("/:channel", h.Get)
	group.POST("/:channel/close", h.Close)
	group.POST("/:channel/activate", h.Activate)
	group.POST("/:channel/renew", h.Renew)
	group.POST("/:channel/bind-identity", h.BindIdentity)
}
func channelFailure(c *gin.Context, e error) {
	api := channels.MapError(e)
	c.JSON(api.Status, api)
}

// channelOutcomeFailure reports a lifecycle refusal with the channel when it exists.
func channelOutcomeFailure(c *gin.Context, result channels.Result) {
	api := channels.MapError(channels.OutcomeError(result.Outcome))
	if result.Channel == nil {
		c.JSON(api.Status, api)
		return
	}
	c.JSON(api.Status, gin.H{"error": api.Message, "code": api.Code, "data": result})
}
func channelInvalid(c *gin.Context, status int, message string) {
	c.JSON(status, communications.APIError{Status: status, Code: "CHANNEL_REQUEST_INVALID", Message: message})
}
func channelIdentity(c *gin.Context) (uuid.UUID, bool) {
	id, e := uuid.Parse(c.Param("channel"))
	// uuid.Parse also accepts URN and braced forms; only the canonical form is an identity.
	if e != nil || id == uuid.Nil || !strings.EqualFold(id.String(), c.Param("channel")) {
		channelFailure(c, channels.ErrIdentifierInvalid)
		return uuid.Nil, false
	}
	return id, true
}
func readChannelBody(c *gin.Context) ([]byte, bool) {
	raw, e := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 4<<10))
	if e != nil {
		channelInvalid(c, 413, "request body exceeds limit")
		return nil, false
	}
	return raw, true
}

// emptyChannelBody accepts an empty body or {} and nothing else.
func emptyChannelBody(c *gin.Context, message string) bool {
	raw, ok := readChannelBody(c)
	if !ok {
		return false
	}
	if trimmed := bytes.TrimSpace(raw); len(trimmed) > 0 {
		var body struct{}
		if e := communications.DecodeStrict(trimmed, &body); e != nil || trimmed[0] != '{' {
			channelInvalid(c, 400, message)
			return false
		}
	}
	return true
}

// Open accepts only {"zone": "Z1"}; identity, lifetime and times come from the database.
func (h *ChannelHandler) Open(c *gin.Context) {
	raw, ok := readChannelBody(c)
	if !ok {
		return
	}
	var body struct {
		Zone *string `json:"zone"`
	}
	if e := communications.DecodeStrict(raw, &body); e != nil || bytes.TrimSpace(raw)[0] != '{' {
		channelInvalid(c, 400, "request must be a JSON object with only a zone")
		return
	}
	if body.Zone == nil {
		channelInvalid(c, 422, "zone is required")
		return
	}
	result, e := h.manager.Open(c.Request.Context(), c.Param("site"), *body.Zone)
	if e != nil {
		channelFailure(c, e)
		return
	}
	c.JSON(201, gin.H{"data": result})
}

// Get never renews. A closed or expired channel is reported by its own status.
func (h *ChannelHandler) Get(c *gin.Context) {
	id, ok := channelIdentity(c)
	if !ok {
		return
	}
	result, e := h.manager.Get(c.Request.Context(), c.Param("site"), id)
	if e != nil {
		channelFailure(c, e)
		return
	}
	if result.Outcome != channels.Found {
		channelOutcomeFailure(c, result)
		return
	}
	c.JSON(200, gin.H{"data": result})
}

// Close takes an empty body (or {}).
func (h *ChannelHandler) Close(c *gin.Context) {
	id, ok := channelIdentity(c)
	if !ok {
		return
	}
	if !emptyChannelBody(c, "close takes no request fields") {
		return
	}
	result, e := h.manager.Close(c.Request.Context(), c.Param("site"), id)
	if e != nil {
		channelFailure(c, e)
		return
	}
	if result.Outcome != channels.Closed {
		channelOutcomeFailure(c, result)
		return
	}
	c.JSON(200, gin.H{"data": result})
}

// Activate protects a pending channel and makes it active. It takes an empty body
// (or {}): the protection comes from CodexHash, CodexTime and the database row,
// never from the request. Repeating it is idempotent (already_active, with the
// first activation's protection).
func (h *ChannelHandler) Activate(c *gin.Context) {
	id, ok := channelIdentity(c)
	if !ok {
		return
	}
	if !emptyChannelBody(c, "activate takes no request fields") {
		return
	}
	result, e := h.manager.Activate(c.Request.Context(), c.Param("site"), id)
	if e != nil {
		channelFailure(c, e)
		return
	}
	switch result.Outcome {
	case channels.Activated, channels.AlreadyActive:
		c.JSON(200, gin.H{"data": result})
	case channels.Closed, channels.Expired, channels.NotFound:
		channelOutcomeFailure(c, result)
	default:
		// The manager reports every other result as an error; never echo it.
		channelFailure(c, errors.New("unexpected activation result"))
	}
}

// Renew moves an active channel's deadline after a visitor communication that the
// site completed through its Seven-Fold Process (Phase C10). It takes an empty body
// (or {}): the deadline comes from database time, never from the request. Pending,
// closed, expired and unknown channels are refused with their lifecycle code.
func (h *ChannelHandler) Renew(c *gin.Context) {
	id, ok := channelIdentity(c)
	if !ok {
		return
	}
	if !emptyChannelBody(c, "renew takes no request fields") {
		return
	}
	result, e := h.manager.Renew(c.Request.Context(), c.Param("site"), id)
	switch {
	case e == nil && result.Outcome == channels.Renewed:
		c.JSON(200, gin.H{"data": result})
	case result.Outcome == channels.Pending || result.Outcome == channels.Closed || result.Outcome == channels.Expired || result.Outcome == channels.NotFound:
		channelOutcomeFailure(c, result)
	case e != nil:
		channelFailure(c, e)
	default:
		channelFailure(c, errors.New("unexpected renewal result"))
	}
}

// BindIdentity links an active guest channel to an authenticated account, once
// (C11-H). The body is exactly {"identity": {"type": "authenticated", "user_uuid":
// "<canonical lowercase UUID>"}}, sent by the site application after its own
// successful login through the trusted gateway. The binding hash comes from
// CodexHash and the database row, never from the request. A refusal reports the
// channel's state and never closes it.
func (h *ChannelHandler) BindIdentity(c *gin.Context) {
	id, ok := channelIdentity(c)
	if !ok {
		return
	}
	raw, ok := readChannelBody(c)
	if !ok {
		return
	}
	var body struct {
		Identity *struct {
			Type     *string `json:"type"`
			UserUUID *string `json:"user_uuid"`
		} `json:"identity"`
	}
	if trimmed := bytes.TrimSpace(raw); len(trimmed) == 0 || trimmed[0] != '{' || communications.DecodeStrict(trimmed, &body) != nil ||
		body.Identity == nil || body.Identity.Type == nil || body.Identity.UserUUID == nil {
		channelInvalid(c, 400, "request must be a JSON object with only an identity type and user_uuid")
		return
	}
	user, e := uuid.Parse(*body.Identity.UserUUID)
	// The canonical lowercase text only; uuid.Parse also accepts other spellings.
	if *body.Identity.Type != "authenticated" || e != nil || user == uuid.Nil || user.String() != *body.Identity.UserUUID {
		channelInvalid(c, 422, "identity must be an authenticated, canonical, non-nil user_uuid")
		return
	}
	result, e := h.manager.BindIdentity(c.Request.Context(), c.Param("site"), id, user)
	if e != nil {
		channelFailure(c, e)
		return
	}
	switch result.Outcome {
	case channels.Bound, channels.AlreadyBound:
		c.JSON(200, gin.H{"data": result})
	case channels.Pending, channels.Closed, channels.Expired, channels.NotFound, channels.IdentityConflict, channels.BindingUnsupported:
		channelOutcomeFailure(c, result)
	default:
		channelFailure(c, errors.New("unexpected identity binding result"))
	}
}
