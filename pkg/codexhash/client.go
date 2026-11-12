package codexhash

// CodexHash API client for CodexSecure.
//
// CodexSecure calls CodexHash for all hashing operations (fingerprints,
// phase-lock digests). When CodexHash is unreachable the client falls back
// to local sha256 so the security pipeline never blocks.
//
// Config env vars:
//   CODEXHASH_API_URL       - base URL (default: http://codexhash-api:8001)
//   CODEXHASH_ENABLED       - "true"/"false" (default: true)
//   CODEXHASH_FAIL_OPEN     - "true"/"false" (default: true)
//   CODEXHASH_TIMEOUT_SEC   - float seconds (default: 3)

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Client is a thin HTTP client for the CodexHash API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	enabled    bool
	failOpen   bool
}

// HashRequest mirrors POST /api/hash/quick
type hashRequest struct {
	Data    string         `json:"data"`
	Mode    string         `json:"mode"`
	Algo    string         `json:"algo"`
	Context map[string]any `json:"context,omitempty"`
}

// HashResponse is the relevant subset of the CodexHash response.
type HashResponse struct {
	// EventHash is the final chained hash produced by CodexHash.
	EventHash string `json:"event_hash"`
	// PayloadHash is the direct sha256 of the submitted data.
	PayloadHash string `json:"payload_hash"`
	// Source indicates whether the hash came from the API or local fallback.
	Source string `json:"source"`
}

// NewClient builds a client from environment variables.
func NewClient() *Client {
	enabled := !strings.EqualFold(strings.TrimSpace(os.Getenv("CODEXHASH_ENABLED")), "false")
	failOpen := !strings.EqualFold(strings.TrimSpace(os.Getenv("CODEXHASH_FAIL_OPEN")), "false")

	timeoutSec := 3.0
	if v := os.Getenv("CODEXHASH_TIMEOUT_SEC"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			timeoutSec = f
		}
	}

	baseURL := strings.TrimRight(os.Getenv("CODEXHASH_API_URL"), "/")
	if baseURL == "" {
		baseURL = "http://codexhash-api:8001"
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

// Hash calls CodexHash POST /api/hash/quick to hash arbitrary data.
// Falls back to local sha256 when CodexHash is disabled or unreachable and fail_open is true.
func (c *Client) Hash(ctx context.Context, data string, context map[string]any) (HashResponse, error) {
	if !c.enabled {
		return c.localFallback(data, "disabled")
	}

	reqBody, err := json.Marshal(hashRequest{
		Data:    data,
		Mode:    "quick",
		Algo:    "sha256",
		Context: context,
	})
	if err != nil {
		return c.fallback(data, fmt.Errorf("marshal: %w", err))
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/hash/quick", bytes.NewReader(reqBody))
	if err != nil {
		return c.fallback(data, fmt.Errorf("build request: %w", err))
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return c.fallback(data, fmt.Errorf("http: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.fallback(data, fmt.Errorf("codexhash status %d", resp.StatusCode))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	if err != nil {
		return c.fallback(data, fmt.Errorf("read body: %w", err))
	}

	var result HashResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return c.fallback(data, fmt.Errorf("decode: %w", err))
	}
	result.Source = "api"
	return result, nil
}

// QuickHash is a convenience wrapper returning just the hash string.
// It uses EventHash from the API, or local sha256 on fallback.
func (c *Client) QuickHash(ctx context.Context, data string) string {
	result, _ := c.Hash(ctx, data, nil)
	if result.EventHash != "" {
		return result.EventHash
	}
	return result.PayloadHash
}

// Health checks whether the CodexHash API is reachable.
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

// fallback decides whether to return a local sha256 or propagate the error.
func (c *Client) fallback(data string, err error) (HashResponse, error) {
	log.Warn().Err(err).Str("base_url", c.baseURL).Msg("CodexHash API unavailable")
	if c.failOpen {
		return c.localFallback(data, "fallback")
	}
	return HashResponse{}, err
}

// localFallback performs sha256 entirely in-process.
func (c *Client) localFallback(data, source string) (HashResponse, error) {
	h := sha256.Sum256([]byte(data))
	digest := hex.EncodeToString(h[:])
	return HashResponse{
		EventHash:   digest,
		PayloadHash: digest,
		Source:      source,
	}, nil
}

// ErrUnavailable reports that CodexHash could not produce a hash. PayloadHash
// never substitutes a local hash, so callers must treat this as a failure.
var ErrUnavailable = errors.New("codexhash unavailable")

// PayloadHashResult is the stateless CodexHash payload hash.
type PayloadHashResult struct {
	PayloadHash string `json:"payload_hash"`
	Algo        string `json:"algo"`
	HashSize    int    `json:"hash_size"`
}

type payloadHashRequest struct {
	Data     string `json:"data"`
	Algo     string `json:"algo"`
	HashSize int    `json:"hash_size"`
}

// PayloadHash asks CodexHash for the payload hash of data (POST /api/hash/payload).
// The operation is stateless: CodexHash creates no event, reads no previous event
// and writes nothing; the value equals the payload_hash of POST /api/hash/ for
// the same data. Unlike Hash, it fails closed: there is no local fallback,
// regardless of CODEXHASH_FAIL_OPEN. The data itself is not logged.
func (c *Client) PayloadHash(ctx context.Context, data string, algo string, hashSize int) (PayloadHashResult, error) {
	if !c.enabled {
		return PayloadHashResult{}, fmt.Errorf("%w: disabled", ErrUnavailable)
	}

	reqBody, err := json.Marshal(payloadHashRequest{
		Data:     data,
		Algo:     algo,
		HashSize: hashSize,
	})
	if err != nil {
		return PayloadHashResult{}, fmt.Errorf("%w: marshal: %v", ErrUnavailable, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/hash/payload", bytes.NewReader(reqBody))
	if err != nil {
		return PayloadHashResult{}, fmt.Errorf("%w: build request: %v", ErrUnavailable, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		log.Warn().Err(err).Str("base_url", c.baseURL).Msg("CodexHash payload hash request failed")
		return PayloadHashResult{}, fmt.Errorf("%w: http: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Warn().Int("status", resp.StatusCode).Str("base_url", c.baseURL).Msg("CodexHash payload hash rejected")
		return PayloadHashResult{}, fmt.Errorf("%w: codexhash status %d", ErrUnavailable, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	if err != nil {
		return PayloadHashResult{}, fmt.Errorf("%w: read body: %v", ErrUnavailable, err)
	}

	var result PayloadHashResult
	if err := json.Unmarshal(body, &result); err != nil {
		return PayloadHashResult{}, fmt.Errorf("%w: decode: %v", ErrUnavailable, err)
	}
	return result, nil
}
