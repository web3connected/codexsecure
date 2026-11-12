package communications

import "testing"

func TestDiscoveryDeclarationValidation(t *testing.T) {
	p := func(v string) *string { return &v }
	v := int64(0)
	base := Discovery{RegistrationKey: "d9fe68fe-b0dc-4aca-b625-56c3514479f2", Scope: "universal", Component: p("main-navigation"), Page: SourcePage{"home", "https://example.test/", "Z1"}, Detected: Detected{"https://example.test/a", "GET", "GET", "fetch", "Z1"}, Declaration: Declaration{p("universal"), p("main-navigation"), p("fetch"), p("products-link")}, BrowserIdentifier: "products", ExpectedStructureRevision: &v}
	s := NewPostgresStore(nil, map[string]bool{"Z1": true})
	if e := s.validateDiscovery(&base); e != nil {
		t.Fatal(e)
	}
	if issues := base.issues(); len(issues) != 0 {
		t.Fatal(issues)
	}
	cases := []struct {
		name   string
		change func(*Discovery)
		issue  string
	}{
		{"missing scope", func(d *Discovery) { d.Declaration.Scope = nil }, "missing_scope"},
		{"invalid scope", func(d *Discovery) { d.Declaration.Scope = p("site") }, "invalid_scope"},
		{"missing component", func(d *Discovery) { d.Declaration.Component = nil }, "missing_component"},
		{"invalid component", func(d *Discovery) { d.Declaration.Component = p("Main Navigation") }, "invalid_component"},
		{"missing item", func(d *Discovery) { d.Declaration.Item = nil }, "missing_item"},
		{"invalid item", func(d *Discovery) { d.Declaration.Item = p("unknown") }, "invalid_item"},
		{"invalid identifier", func(d *Discovery) { d.Declaration.ItemIdentifier = p("Products Link") }, "invalid_item_identifier"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := base
			tc.change(&d)
			found := false
			for _, issue := range d.issues() {
				if issue == tc.issue {
					found = true
				}
			}
			if !found {
				t.Fatal(d.issues())
			}
		})
	}
	for _, tc := range []struct {
		name   string
		change func(*Discovery)
	}{
		{"non UUID identity", func(d *Discovery) { d.RegistrationKey = "cs-item1:declaration" }},
		{"invalid ownership", func(d *Discovery) { d.Scope = "site" }},
		{"relative observation", func(d *Discovery) { d.Detected.Destination = "/a" }},
		{"missing version", func(d *Discovery) { d.ExpectedStructureRevision = nil }},
		{"invalid method", func(d *Discovery) { d.Detected.Method = "TRACE" }},
		{"unsupported transport", func(d *Discovery) { d.Detected.TransportMethod = "TRACE" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := base
			tc.change(&d)
			if e := s.validateDiscovery(&d); e == nil {
				t.Fatal("invalid discovery accepted")
			}
		})
	}
}
func TestStoredReadDocumentValidation(t *testing.T) {
	s := NewPostgresStore(nil, map[string]bool{"Z1": true})
	raw := []byte(`{"document":{"schema_version":1,"destination":"/a","destination_match":"exact","methods":["GET"],"type":"fetch","zone":"Z1","enabled":true,"unknown":true}}`)
	if e := s.validateEmbeddedDocument(raw); e == nil {
		t.Fatal("stored document unknown field was ignored")
	}
}

func TestObservedBehaviorIsNotValidatedAsApprovedPolicy(t *testing.T) {
	s := NewPostgresStore(nil, map[string]bool{"Z1": true})
	for _, tc := range []struct{ kind, method string }{{"link", "POST"}, {"form", "HEAD"}, {"fetch", "DELETE"}} {
		t.Run(tc.kind+tc.method, func(t *testing.T) {
			revision := int64(0)
			d := Discovery{RegistrationKey: "d9fe68fe-b0dc-4aca-b625-56c3514479f2", Scope: "page", Page: SourcePage{"home", "https://example.test/", "Z1"}, Detected: Detected{"https://example.test/a", tc.method, "POST", tc.kind, "Z1"}, BrowserIdentifier: "item", ExpectedStructureRevision: &revision}
			if e := s.validateDiscovery(&d); e != nil {
				t.Fatalf("observed evidence rejected by policy rules: %v", e)
			}
			raw := []byte(`{"schema_version":1,"destination":"/a","destination_match":"exact","methods":["` + tc.method + `"],"type":"` + tc.kind + `","zone":"Z1","enabled":true}`)
			_, e := Validate(raw, s.zones)
			if tc.kind != "fetch" && e == nil {
				t.Fatal("approval policy type restrictions were weakened")
			}
		})
	}
}
