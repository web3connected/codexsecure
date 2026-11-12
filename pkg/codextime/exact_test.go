package codextime

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func exactServer(t *testing.T, status int, body string) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/time/current-tiu" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	t.Setenv("CODEXTIME_API_URL", server.URL)
}

func legacyBody(tiu, cached string) string {
	return `{"tiu":` + tiu + `,"timestamp":"2026-10-19T22:36:14.695801Z","epoch":"2020-01-01 00:00:00","cached":` + cached + `}`
}

// The TIU text is kept exactly as written on the wire, padded only to six decimals.
func TestCurrentTIUExactPreservesWireText(t *testing.T) {
	for wire, want := range map[string]string{
		"214612574.695765":      "214612574.695765",
		"214612574.5":           "214612574.500000",
		"214612574.0":           "214612574.000000",
		"214612574":             "214612574.000000",
		"0.000001":              "0.000001",
		"99999999999999.999999": "99999999999999.999999", // 20 digits: not representable in float64
		"12345678901234.123457": "12345678901234.123457",
	} {
		exactServer(t, 200, legacyBody(wire, "false"))
		got, err := NewClient().CurrentTIUExact(context.Background())
		if err != nil || got != (ExactTIU{Value: want, Scheme: LegacySecondsScheme, Unit: LegacySecondsUnit}) {
			t.Fatalf("%s: got %+v, %v", wire, got, err)
		}
	}
}

func TestCurrentTIUExactRejectsInvalidResponses(t *testing.T) {
	for name, body := range map[string]string{
		"string tiu":        legacyBody(`"214612574.695765"`, "false"),
		"zero":              legacyBody("0", "false"),
		"zero fraction":     legacyBody("0.000000", "false"),
		"negative":          legacyBody("-1.5", "false"),
		"exponent":          legacyBody("2.146e8", "false"),
		"seven decimals":    legacyBody("214612574.6957651", "false"),
		"leading zero":      legacyBody("0214612574.5", "false"),
		"too many digits":   legacyBody("123456789012345.5", "false"),
		"null tiu":          legacyBody("null", "false"),
		"missing tiu":       `{"timestamp":"2026-10-19T22:36:14Z","epoch":"2020-01-01 00:00:00","cached":false}`,
		"missing cached":    `{"tiu":1.5,"timestamp":"2026-10-19T22:36:14Z","epoch":"2020-01-01 00:00:00"}`,
		"string cached":     legacyBody("1.5", `"false"`),
		"missing timestamp": `{"tiu":1.5,"epoch":"2020-01-01 00:00:00","cached":false}`,
		"other epoch":       `{"tiu":1.5,"timestamp":"2026-10-19T22:36:14Z","epoch":"1970-01-01 00:00:00","cached":false}`,
		"missing epoch":     `{"tiu":1.5,"timestamp":"2026-10-19T22:36:14Z","cached":false}`,
		"duplicate tiu":     `{"tiu":1.5,"tiu":2.5,"timestamp":"2026-10-19T22:36:14Z","epoch":"2020-01-01 00:00:00","cached":false}`,
		"trailing data":     legacyBody("1.5", "false") + `{}`,
		"array":             `[1.5]`,
		"not json":          `tiu=1.5`,
		"empty":             ``,
		"fail-open shape":   `{"tiu":0,"source":"unavailable"}`,
		"v1 snapshot":       `{"contract":"codextime-tiu-api/1","values":{"ns":"214612574695765961"}}`,
		"raw NUL bytes":     legacyBody("1.5", "false")[:20] + string(make([]byte, 5000)),
	} {
		exactServer(t, 200, body)
		got, err := NewClient().CurrentTIUExact(context.Background())
		if !errors.Is(err, ErrTIUInvalid) || got != (ExactTIU{}) {
			t.Fatalf("%s: got %+v, %v", name, got, err)
		}
	}
}

// A cached TIU is never accepted, and is not reported as malformed.
func TestCurrentTIUExactRejectsCached(t *testing.T) {
	exactServer(t, 200, legacyBody("214612574.695765", "true"))
	got, err := NewClient().CurrentTIUExact(context.Background())
	if !errors.Is(err, ErrTIUCached) || errors.Is(err, ErrTIUInvalid) || got != (ExactTIU{}) {
		t.Fatalf("got %+v, %v", got, err)
	}
}

// Unlike CurrentTIU, the exact read never fails open or substitutes a value.
func TestCurrentTIUExactFailsClosed(t *testing.T) {
	t.Setenv("CODEXTIME_FAIL_OPEN", "true")
	exactServer(t, 500, legacyBody("214612574.695765", "false"))
	if got, err := NewClient().CurrentTIUExact(context.Background()); !errors.Is(err, ErrTIUUnavailable) || got != (ExactTIU{}) {
		t.Fatalf("status 500: got %+v, %v", got, err)
	}
	t.Setenv("CODEXTIME_API_URL", "http://127.0.0.1:1")
	if got, err := NewClient().CurrentTIUExact(context.Background()); !errors.Is(err, ErrTIUUnavailable) || got != (ExactTIU{}) {
		t.Fatalf("unreachable: got %+v, %v", got, err)
	}
	exactServer(t, 200, legacyBody("214612574.695765", "false"))
	t.Setenv("CODEXTIME_ENABLED", "false")
	if got, err := NewClient().CurrentTIUExact(context.Background()); !errors.Is(err, ErrTIUUnavailable) || got != (ExactTIU{}) {
		t.Fatalf("disabled: got %+v, %v", got, err)
	}
}
