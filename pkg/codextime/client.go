package codextime

// CodexTime API client for CodexSecure.
//
// CodexSecure calls CodexTime to fetch the current live TIU when
// TIU_HMAC_SECRET is not set (offline signed-token mode is disabled).
//
// Config env vars:
//   CODEXTIME_API_URL       - base URL (default: http://codextime-api:50002)
//   CODEXTIME_ENABLED       - "true"/"false" (default: true)
//   CODEXTIME_FAIL_OPEN     - "true"/"false" (default: true)
//   CODEXTIME_TIMEOUT_SEC   - float seconds (default: 3)

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// TIUResponse is the subset we need from CodexTime GET /api/time/current-tiu.
type TIUResponse struct {
	TIU       float64 `json:"tiu"`
	Timestamp string  `json:"timestamp"`
	Epoch     string  `json:"epoch"`
	Cached    bool    `json:"cached"`
	// Source indicates "api" or "unavailable".
	Source string `json:"source"`
}

// Client is a thin HTTP client for the CodexTime API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	enabled    bool
	failOpen   bool
}

// NewClient builds a client from environment variables.
func NewClient() *Client {
	enabled := !strings.EqualFold(strings.TrimSpace(os.Getenv("CODEXTIME_ENABLED")), "false")
	failOpen := !strings.EqualFold(strings.TrimSpace(os.Getenv("CODEXTIME_FAIL_OPEN")), "false")

	timeoutSec := 3.0
	if v := os.Getenv("CODEXTIME_TIMEOUT_SEC"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			timeoutSec = f
		}
	}

	baseURL := strings.TrimRight(os.Getenv("CODEXTIME_API_URL"), "/")
	if baseURL == "" {
		baseURL = "http://codextime-api:50002"
	}

	return &Client{
		baseURL:  baseURL,
		enabled:  enabled,
		failOpen: failOpen,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec * float64(time.Second)),
		},
	}
}

// CurrentTIU fetches the live TIU from CodexTime.
// Falls back gracefully (returns TIU=0, Source="unavailable") when fail_open is true.
func (c *Client) CurrentTIU(ctx context.Context) (TIUResponse, error) {
	if !c.enabled {
		return TIUResponse{Source: "disabled"}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/time/current-tiu", nil)
	if err != nil {
		return c.fallback(fmt.Errorf("build request: %w", err))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.fallback(fmt.Errorf("http: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.fallback(fmt.Errorf("codextime status %d", resp.StatusCode))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
	if err != nil {
		return c.fallback(fmt.Errorf("read body: %w", err))
	}

	var result TIUResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return c.fallback(fmt.Errorf("decode: %w", err))
	}
	result.Source = "api"
	return result, nil
}

// Health checks whether the CodexTime API is reachable.
func (c *Client) Health(ctx context.Context) bool {
	if !c.enabled {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 300
}

func (c *Client) fallback(err error) (TIUResponse, error) {
	log.Warn().Err(err).Str("base_url", c.baseURL).Msg("CodexTime API unavailable")
	if c.failOpen {
		return TIUResponse{Source: "unavailable"}, nil
	}
	return TIUResponse{}, err
}
