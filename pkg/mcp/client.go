package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// Client represents MCP server HTTP client
type Client struct {
	baseURL    string
	httpClient *http.Client
	enabled    bool
}

// NewClient creates a new MCP client
func NewClient(baseURL string, enabled bool) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		enabled: enabled,
	}
}

// IngressEvaluationRequest represents the Stage 1 ingress evaluation request to MCP.
type IngressEvaluationRequest struct {
	Path    string            `json:"path"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Payload any               `json:"payload,omitempty"`
}

// IngressEvaluationResponse represents the Stage 1 MCP response.
type IngressEvaluationResponse struct {
	IngressPathZone string   `json:"ingress_path_zone"`
	IngressRole     string   `json:"ingress_role"`
	ProposedZoneID  string   `json:"proposed_zone_id"`
	ProposedZoneName string  `json:"proposed_zone_name"`
	Confidence      float64  `json:"confidence"`
	DetectionMethod string   `json:"detection_method"`
	QuarantineCandidate bool `json:"quarantine_candidate"`
	QuarantineReason string  `json:"quarantine_reason,omitempty"`
	Z0Directive     string   `json:"z0_directive,omitempty"`
	Z0Reason        string   `json:"z0_reason,omitempty"`
	Z0ConfidenceGate float64 `json:"z0_confidence_gate,omitempty"`
	ExpectedEnvelope struct {
		MaxResponseTimeMS  int     `json:"max_response_time_ms"`
		RequiredAuth       bool    `json:"required_auth"`
		RequiredDeviceTrust float64 `json:"required_device_trust"`
	} `json:"expected_envelope"`
}

// EvaluateIngress sends a Stage 1 ingress evaluation request to MCP.
func (c *Client) EvaluateIngress(ctx context.Context, req IngressEvaluationRequest) (*IngressEvaluationResponse, error) {
	if !c.enabled {
		log.Debug().Msg("MCP integration disabled, skipping ingress evaluation")
		return nil, nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/ingress/evaluate", c.baseURL),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("MCP returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result IngressEvaluationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Health checks MCP server health
func (c *Client) Health(ctx context.Context) error {
	if !c.enabled {
		return nil
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/health", c.baseURL),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MCP health check returned status %d", resp.StatusCode)
	}

	return nil
}

// IsEnabled returns whether MCP integration is enabled
func (c *Client) IsEnabled() bool {
	return c.enabled
}
