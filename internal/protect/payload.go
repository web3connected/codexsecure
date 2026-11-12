// Package protect prepares communication payloads for the Stage 4 (Protect)
// CodexHash and TIU services.
package protect

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Representation names the canonical payload form hashed by CodexHash.
//
// codexsecure-protect-payload-v1:
//   - the request's "payload" value, which must be a non-empty JSON object
//   - object keys sorted by byte order at every depth; array order preserved
//   - numbers kept exactly as written in the request (no reformatting)
//   - compact JSON, no insignificant whitespace
//   - strings re-encoded without escaping "/", "&", "<", ">" or non-ASCII
//     (U+2028/U+2029 are escaped, as by PHP json_encode with
//     JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE)
//   - duplicate keys are rejected
//   - top-level "codexhash" and "tiu" are rejected, so neither value can be
//     part of its own input
const Representation = "codexsecure-protect-payload-v1"

// ReservedFields cannot appear at the top level of a payload.
var ReservedFields = []string{"codexhash", "tiu"}

var (
	ErrInvalidPayload = errors.New("invalid payload")
	ErrReservedField  = errors.New("reserved field")
	ErrDuplicateKey   = errors.New("duplicate key")
)

// CanonicalPayload validates a raw payload and returns its canonical form.
func CanonicalPayload(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return "", fmt.Errorf("%w: payload must be a JSON object", ErrInvalidPayload)
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()

	var out strings.Builder
	keys, err := writeValue(decoder, &out, true)
	if err != nil {
		return "", err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return "", fmt.Errorf("%w: trailing data after payload", ErrInvalidPayload)
	}
	if keys == 0 {
		return "", fmt.Errorf("%w: payload must not be empty", ErrInvalidPayload)
	}
	return out.String(), nil
}

// writeValue writes the next JSON value in canonical form. For objects it
// returns the number of keys.
func writeValue(decoder *json.Decoder, out *strings.Builder, topLevel bool) (int, error) {
	token, err := decoder.Token()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}

	switch value := token.(type) {
	case json.Delim:
		switch value {
		case '{':
			return writeObject(decoder, out, topLevel)
		case '[':
			return 0, writeArray(decoder, out)
		}
		return 0, fmt.Errorf("%w: unexpected %q", ErrInvalidPayload, value)
	case string:
		out.WriteString(encodeString(value))
	case json.Number:
		out.WriteString(value.String())
	case bool:
		if value {
			out.WriteString("true")
		} else {
			out.WriteString("false")
		}
	case nil:
		out.WriteString("null")
	default:
		return 0, fmt.Errorf("%w: unsupported value", ErrInvalidPayload)
	}
	return 0, nil
}

func writeObject(decoder *json.Decoder, out *strings.Builder, topLevel bool) (int, error) {
	members := map[string]string{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return 0, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
		}
		key, ok := token.(string)
		if !ok {
			return 0, fmt.Errorf("%w: object key must be a string", ErrInvalidPayload)
		}
		if _, exists := members[key]; exists {
			return 0, fmt.Errorf("%w: %q", ErrDuplicateKey, key)
		}
		if topLevel {
			for _, reserved := range ReservedFields {
				if key == reserved {
					return 0, fmt.Errorf("%w: %q must not be part of the payload", ErrReservedField, key)
				}
			}
		}

		var member strings.Builder
		if _, err := writeValue(decoder, &member, false); err != nil {
			return 0, err
		}
		members[key] = member.String()
	}
	if _, err := decoder.Token(); err != nil { // closing '}'
		return 0, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}

	keys := make([]string, 0, len(members))
	for key := range members {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out.WriteByte('{')
	for i, key := range keys {
		if i > 0 {
			out.WriteByte(',')
		}
		out.WriteString(encodeString(key))
		out.WriteByte(':')
		out.WriteString(members[key])
	}
	out.WriteByte('}')
	return len(keys), nil
}

func writeArray(decoder *json.Decoder, out *strings.Builder) error {
	out.WriteByte('[')
	for i := 0; decoder.More(); i++ {
		if i > 0 {
			out.WriteByte(',')
		}
		if _, err := writeValue(decoder, out, false); err != nil {
			return err
		}
	}
	if _, err := decoder.Token(); err != nil { // closing ']'
		return fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	out.WriteByte(']')
	return nil
}

func encodeString(value string) string {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
	return strings.TrimSuffix(buffer.String(), "\n")
}
