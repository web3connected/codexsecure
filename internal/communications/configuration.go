// Package communications implements administrator-owned configuration policy.
// Discovery values never modify these documents or grant authorization.
package communications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Document struct {
	SchemaVersion    int      `json:"schema_version"`
	Destination      string   `json:"destination"`
	DestinationMatch string   `json:"destination_match"`
	Methods          []string `json:"methods"`
	Type             string   `json:"type"`
	Zone             string   `json:"zone"`
	Enabled          bool     `json:"enabled"`
	// Structure is the approved structural identity of a form (HTTP Codex7FP Stage 3), optional in
	// schema version 1. Laravel computes it (CodexHash v2 through the CodexSecure service) when it
	// prepares the draft; it is validated here for form only and never computed or interpreted.
	Structure *StructureBinding `json:"structure,omitempty"`
}

// StructureSchema marks the canonical, values-free structure document the binding's hash covers.
const StructureSchema = "codexsecure-communication-structure-v1"

// StructureBinding is a form's approved Form ID, field structure (names, parts, shapes and presence;
// never values) and the CodexHash v2 digest of schema + Form ID + always-submitted fields.
type StructureBinding struct {
	Schema    string           `json:"schema"`
	FormID    string           `json:"form_id"`
	Fields    []StructureField `json:"fields"`
	CodexHash string           `json:"codex_hash"`
}
type StructureField struct {
	Part     string `json:"part"`
	Path     string `json:"path"`
	Shape    string `json:"shape"`
	Presence string `json:"presence"`
}

var (
	structureHash   = regexp.MustCompile(`^ch2:inv-dual/1:512:[0-9a-f]{128}$`)
	structureFormID = regexp.MustCompile(`^\S{1,256}$`)
)

type ValidationErrors map[string][]string

func (e ValidationErrors) Error() string { return "configuration does not satisfy schema version 1" }

var methodOrder = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

// DecodeStrict also rejects duplicate keys, missing/null fields, and trailing JSON.
func DecodeStrict(raw []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	if err := uniqueKeys(d); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("expected one JSON value")
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	return d.Decode(target)
}
func uniqueKeys(d *json.Decoder) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			t, e := d.Token()
			if e != nil {
				return e
			}
			k, ok := t.(string)
			if !ok || seen[k] {
				return fmt.Errorf("duplicate or invalid JSON key")
			}
			seen[k] = true
			if e = uniqueKeys(d); e != nil {
				return e
			}
		}
	case '[':
		for d.More() {
			if e := uniqueKeys(d); e != nil {
				return e
			}
		}
	default:
		return fmt.Errorf("invalid JSON")
	}
	_, err = d.Token()
	return err
}

func Validate(raw []byte, zones map[string]bool) (Document, error) {
	var result Document
	errors := ValidationErrors{}
	var fields map[string]json.RawMessage
	if err := DecodeStrict(raw, &fields); err != nil || fields == nil {
		return result, ValidationErrors{"document": {"must be a JSON object"}}
	}
	required := []string{"schema_version", "destination", "destination_match", "methods", "type", "zone", "enabled"}
	allowed := map[string]bool{}
	for _, k := range required {
		allowed[k] = true
		if v, ok := fields[k]; !ok || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			errors[k] = []string{"is required and cannot be null"}
		}
	}
	allowed["structure"] = true
	for k := range fields {
		if !allowed[k] {
			errors[k] = []string{"is not part of schema version 1"}
		}
	}
	if len(errors) > 0 {
		return result, errors
	}
	if err := DecodeStrict(raw, &result); err != nil {
		return result, ValidationErrors{"document": {"invalid field type"}}
	}
	if result.SchemaVersion != 1 {
		errors["schema_version"] = []string{"must be 1"}
	}
	if !ValidDestination(result.Destination) {
		errors["destination"] = []string{"must be an application path or HTTP(S) URL without credentials, fragment, whitespace or backslashes"}
	}
	if result.DestinationMatch != "exact" {
		errors["destination_match"] = []string{"must be exact"}
	}
	if result.Type == "ajax" {
		result.Type = "fetch"
	}
	if result.Type != "link" && result.Type != "form" && result.Type != "fetch" {
		errors["type"] = []string{"must be link, form or fetch"}
	}
	if !zones[result.Zone] {
		errors["zone"] = []string{"must exist in the zone registry"}
	}
	if len(result.Methods) == 0 {
		errors["methods"] = []string{"select at least one HTTP method"}
	}
	seen := map[string]bool{}
	for _, m := range result.Methods {
		m = strings.ToUpper(strings.Trim(m, " \t\n\r\x00\v"))
		supported := false
		for _, a := range methodOrder {
			if m == a {
				supported = true
			}
		}
		if !supported || seen[m] {
			errors["methods"] = []string{"methods must be supported and unique after normalization"}
		}
		if result.Type == "link" && m != "GET" || result.Type == "form" && (m == "HEAD" || m == "OPTIONS") {
			errors["methods"] = []string{"method not supported by communication type"}
		}
		seen[m] = true
	}
	result.Methods = []string{}
	for _, m := range methodOrder {
		if seen[m] {
			result.Methods = append(result.Methods, m)
		}
	}
	if raw, present := fields["structure"]; present {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			errors["structure"] = []string{"cannot be null; omit it when no structure is bound"}
		} else if problem := validStructure(result.Type, result.Structure); problem != "" {
			errors["structure"] = []string{problem}
		}
	}
	if len(errors) > 0 {
		return result, errors
	}
	return result, nil
}

// validStructure applies the SQL validator's rules for the optional structure binding.
func validStructure(kind string, b *StructureBinding) string {
	if kind != "form" {
		return "only a form carries an approved structure"
	}
	if b.Schema != StructureSchema {
		return "schema must be " + StructureSchema
	}
	if !structureFormID.MatchString(b.FormID) || utf8.RuneCountInString(b.FormID) > 256 {
		return "form_id must be 1 to 256 characters without whitespace"
	}
	if !structureHash.MatchString(b.CodexHash) {
		return "codex_hash must be a CodexHash v2 inv-dual/1 512-bit digest"
	}
	if b.Fields == nil {
		return "fields must be a list"
	}
	seen := map[string]bool{}
	for _, f := range b.Fields {
		identity := f.Part + "\x1f" + f.Path
		if f.Part != "form" && f.Part != "file" || f.Path == "" || utf8.RuneCountInString(f.Path) > 1024 || seen[identity] ||
			f.Shape != "value" && f.Shape != "file" && f.Shape != "container" || f.Presence != "always" && f.Presence != "conditional" {
			return "each field needs a form or file part, a unique path of 1 to 1024 characters, a value, file or container shape and always or conditional presence"
		}
		seen[identity] = true
	}
	return ""
}

func ValidDestination(value string) bool {
	if value == "" || !utf8.ValidString(value) || utf8.RuneCountInString(value) > 2048 {
		return false
	}
	for _, r := range value {
		if unicode.IsSpace(r) || r == '\\' || r == '#' || unicode.IsControl(r) {
			return false
		}
	}
	if strings.HasPrefix(value, "/") {
		return !strings.HasPrefix(value, "//")
	}
	u, err := url.Parse(value)
	return err == nil && (strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https")) && u.Hostname() != "" && u.User == nil && u.Fragment == ""
}
