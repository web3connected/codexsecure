package codextime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

// TIU schemes and units as the codex_zones channel protection contract (C3A)
// names them. The interim legacy service issues seconds since 2020-01-01 with
// six decimals; CodexTime V1 (codextime-tiu-2020/1) is not served by this client.
const (
	LegacySecondsScheme = "codextime-legacy-seconds-2020/us"
	LegacySecondsUnit   = "s"
	legacyEpoch         = "2020-01-01 00:00:00"
)

var (
	// ErrTIUUnavailable: CodexTime could not be asked, or did not answer with 2xx.
	ErrTIUUnavailable = errors.New("codextime tiu unavailable")
	// ErrTIUCached: CodexTime answered with a cached TIU, which is never accepted.
	ErrTIUCached = errors.New("codextime tiu is cached")
	// ErrTIUInvalid: the response is not a TIU in a supported scheme.
	ErrTIUInvalid = errors.New("codextime tiu response is invalid")
)

// ExactTIU is a TIU as exact decimal text with its scheme and unit. It never
// passes through float64.
type ExactTIU struct {
	Value  string
	Scheme string
	Unit   string
}

// The JSON number text the legacy service writes for round(tiu, 6): no sign,
// exponent or leading zeros, at most six decimals.
var legacyWireTIU = regexp.MustCompile(`^(0|[1-9][0-9]{0,13})(\.[0-9]{1,6})?$`)

// CurrentTIUExact reads GET /api/time/current-tiu losslessly and fails closed:
// CODEXTIME_FAIL_OPEN is ignored, a disabled client is an error, and there is no
// fallback, zero or cached value. The TIU's number text is kept exactly and only
// padded with trailing zeros to the scheme's six decimals; more decimals are
// rejected, never rounded. CodexTime's calculation is not re-evaluated here.
func (c *Client) CurrentTIUExact(ctx context.Context) (ExactTIU, error) {
	if !c.enabled {
		return ExactTIU{}, fmt.Errorf("%w: disabled", ErrTIUUnavailable)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/time/current-tiu", nil)
	if err != nil {
		return ExactTIU{}, fmt.Errorf("%w: build request: %v", ErrTIUUnavailable, err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Warn().Err(err).Str("base_url", c.baseURL).Msg("CodexTime exact TIU request failed")
		return ExactTIU{}, fmt.Errorf("%w: http: %v", ErrTIUUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Warn().Int("status", resp.StatusCode).Str("base_url", c.baseURL).Msg("CodexTime exact TIU rejected")
		return ExactTIU{}, fmt.Errorf("%w: codextime status %d", ErrTIUUnavailable, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
	if err != nil {
		return ExactTIU{}, fmt.Errorf("%w: read body: %v", ErrTIUUnavailable, err)
	}
	return parseLegacyTIU(body)
}

func parseLegacyTIU(body []byte) (ExactTIU, error) {
	fields, err := topLevelFields(body)
	if err != nil {
		return ExactTIU{}, fmt.Errorf("%w: %v", ErrTIUInvalid, err)
	}
	var epoch string
	var timestamp string
	if json.Unmarshal(fields["epoch"], &epoch) != nil || epoch != legacyEpoch {
		return ExactTIU{}, fmt.Errorf("%w: unsupported epoch", ErrTIUInvalid)
	}
	if json.Unmarshal(fields["timestamp"], &timestamp) != nil || timestamp == "" {
		return ExactTIU{}, fmt.Errorf("%w: missing timestamp", ErrTIUInvalid)
	}
	if raw := fields["cached"]; !bytes.Equal(raw, []byte("false")) {
		if bytes.Equal(raw, []byte("true")) {
			return ExactTIU{}, ErrTIUCached
		}
		return ExactTIU{}, fmt.Errorf("%w: cached must be a boolean", ErrTIUInvalid)
	}
	// The raw JSON number text, never a decoded number.
	value := string(fields["tiu"])
	match := legacyWireTIU.FindStringSubmatch(value)
	if match == nil {
		return ExactTIU{}, fmt.Errorf("%w: tiu is not a supported decimal number", ErrTIUInvalid)
	}
	fraction := strings.TrimPrefix(match[2], ".")
	value = match[1] + "." + fraction + strings.Repeat("0", 6-len(fraction))
	if strings.Trim(value, "0.") == "" {
		return ExactTIU{}, fmt.Errorf("%w: tiu must be positive", ErrTIUInvalid)
	}
	return ExactTIU{Value: value, Scheme: LegacySecondsScheme, Unit: LegacySecondsUnit}, nil
}

// topLevelFields returns the raw members of one JSON object, rejecting duplicate
// keys and trailing data.
func topLevelFields(body []byte) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(body))
	if t, err := d.Token(); err != nil || t != json.Delim('{') {
		return nil, errors.New("response must be a JSON object")
	}
	fields := map[string]json.RawMessage{}
	for d.More() {
		t, err := d.Token()
		if err != nil {
			return nil, err
		}
		key, _ := t.(string)
		if _, seen := fields[key]; seen {
			return nil, fmt.Errorf("duplicate key %q", key)
		}
		var raw json.RawMessage
		if err := d.Decode(&raw); err != nil {
			return nil, err
		}
		fields[key] = raw
	}
	if _, err := d.Token(); err != nil {
		return nil, err
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, errors.New("trailing data after response")
	}
	return fields, nil
}
