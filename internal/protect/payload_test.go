package protect

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Computed independently in Python for testdata/protect/laravel_get_payload.json:
// json.dumps(payload, sort_keys=True, separators=(",", ":"), ensure_ascii=False)
const (
	laravelCanonical = `{"communication_id":"8f14e45f-ceea-467a-9b1e-5d2c3a1b7e10","communication_metadata":{"evidence":[],"intent":"Page d'accueil — public & open","received_at":"2026-10-11T20:00:00+00:00"},"declared_zone":"Z1","received_request":{"method":"GET","path":"/","payload":{"body":[],"content_type":null,"form":{},"query":{}}},"request":{"method":"GET","path":"/"},"route":{"action":"__invoke","controller":"App\\Http\\Controllers\\CodexHomeController","methods":["GET","HEAD"],"name":"home","uri":"/"},"route_validation":{"checks":3,"status":"valid"}}`
	laravelSHA512    = "7d225e21ff80847f8d612b011c208fb15e1f519829c5243e22b7de2339968c122d4c8148f3a9a6b1846d5da23c0e099aa3e5e7941aabb263a078bdf8f4d628b0"
)

func TestCanonicalPayloadMatchesIndependentImplementation(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "testdata", "protect", "laravel_get_payload.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var envelope struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(content, &envelope); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	canonical, err := CanonicalPayload(envelope.Payload)
	if err != nil {
		t.Fatalf("canonical payload: %v", err)
	}
	if canonical != laravelCanonical {
		t.Fatalf("canonical form differs\n got: %s\nwant: %s", canonical, laravelCanonical)
	}

	sum := sha512.Sum512([]byte(canonical))
	if got := hex.EncodeToString(sum[:]); got != laravelSHA512 {
		t.Errorf("sha512 = %s, want %s", got, laravelSHA512)
	}
}

func TestCanonicalPayloadIsStableForEquivalentPayloads(t *testing.T) {
	a := `{"zone":"z1","route":"example.route","method":"GET","communication_uuid":"u-1","meta":{"b":2,"a":[3,1]}}`
	b := "{\n  \"meta\": {\"a\": [3, 1], \"b\": 2},\n  \"communication_uuid\": \"u-1\",\n  \"method\": \"GET\",\n  \"route\": \"example.route\",\n  \"zone\": \"z1\"\n}"

	ca, errA := CanonicalPayload(json.RawMessage(a))
	cb, errB := CanonicalPayload(json.RawMessage(b))
	if errA != nil || errB != nil {
		t.Fatalf("errors: %v, %v", errA, errB)
	}
	if ca != cb {
		t.Fatalf("equivalent payloads differ:\n%s\n%s", ca, cb)
	}
	if want := `{"communication_uuid":"u-1","meta":{"a":[3,1],"b":2},"method":"GET","route":"example.route","zone":"z1"}`; ca != want {
		t.Errorf("canonical = %s, want %s", ca, want)
	}

	// Array order and values are significant.
	cc, _ := CanonicalPayload(json.RawMessage(`{"meta":{"a":[1,3],"b":2},"communication_uuid":"u-1","method":"GET","route":"example.route","zone":"z1"}`))
	if cc == ca {
		t.Error("different array order must produce a different representation")
	}
}

func TestCanonicalPayloadPreservesValues(t *testing.T) {
	canonical, err := CanonicalPayload(json.RawMessage(`{"n":[1.0,1e3,-0,10000000000000000000001],"s":"a\/b é <x> & y","e":"\u2028","u":null,"t":true,"f":false,"o":{},"l":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"e":"\u2028","f":false,"l":[],"n":[1.0,1e3,-0,10000000000000000000001],"o":{},"s":"a/b é <x> & y","t":true,"u":null}`
	if canonical != want {
		t.Errorf("canonical = %q\nwant      %q", canonical, want)
	}
}

func TestCanonicalPayloadRejectsInvalidPayloads(t *testing.T) {
	cases := map[string]struct {
		raw  string
		want error
	}{
		"not an object":        {`["a"]`, ErrInvalidPayload},
		"string":               {`"payload"`, ErrInvalidPayload},
		"null":                 {`null`, ErrInvalidPayload},
		"empty object":         {`{}`, ErrInvalidPayload},
		"malformed":            {`{"a":`, ErrInvalidPayload},
		"trailing data":        {`{"a":1} {"b":2}`, ErrInvalidPayload},
		"duplicate key":        {`{"a":1,"a":2}`, ErrDuplicateKey},
		"nested duplicate key": {`{"a":{"b":1,"b":1}}`, ErrDuplicateKey},
		"codexhash in payload": {`{"route":"r","codexhash":"x"}`, ErrReservedField},
		"tiu in payload":       {`{"route":"r","tiu":213912663.4}`, ErrReservedField},
	}
	for name, tc := range cases {
		if _, err := CanonicalPayload(json.RawMessage(tc.raw)); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", name, err, tc.want)
		}
	}
}

// The boundary is the top level, where the SDK attaches security values.
// Nested application data may legitimately use these words.
func TestCanonicalPayloadAllowsReservedWordsBelowTopLevel(t *testing.T) {
	if _, err := CanonicalPayload(json.RawMessage(`{"received_request":{"query":{"tiu":"1"}}}`)); err != nil {
		t.Errorf("nested tiu key rejected: %v", err)
	}
}
