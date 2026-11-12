//go:build integration

package communications

import (
	"encoding/json"
	"github.com/google/uuid"
	"testing"
)

func TestStage2PageIdentityAndEmpty(t *testing.T) {
	ctx, s, admin, site := disposableStore(t)
	_, e := admin.Exec(ctx, `INSERT INTO pages(site_id,page_key,url,zone) SELECT id,'empty','https://example.test/pages/empty','Z2' FROM sites WHERE site_key=$1`, site)
	if e != nil {
		t.Fatal(e)
	}
	a, e := s.Assess(ctx, site, Coverage{Type: "page", Page: "https://example.test/pages/empty"})
	if e != nil || a.Page == nil || *a.Page != (SourcePage{"empty", "https://example.test/pages/empty", "Z2"}) || a.Communications == nil || len(a.Communications) != 0 {
		t.Fatal(a, e)
	}
	raw, e := json.Marshal(a)
	if e != nil {
		t.Fatal(e)
	}
	var payload map[string]any
	if e = json.Unmarshal(raw, &payload); e != nil {
		t.Fatal(e)
	}
	if list, ok := payload["communications"].([]any); !ok || len(list) != 0 {
		t.Fatal(string(raw))
	}
	for _, url := range []string{"https://example.test/pages/empty/", "https://example.test/pages/empty?q=1", "https://example.test/pages/Empty", "https://example.test/empty"} {
		if _, e = s.Assess(ctx, site, Coverage{Type: "page", Page: url}); MapError(e).Code != "PAGE_NOT_FOUND" {
			t.Fatalf("%s: %v", url, e)
		}
	}
	for _, url := range []string{"/pages/empty", "https://unrelated.test/pages/empty"} {
		if _, e = s.Assess(ctx, site, Coverage{Type: "page", Page: url}); MapError(e).Status != 422 {
			t.Fatal(url, e)
		}
	}
	if _, e = s.Assess(ctx, site+"-unknown", Coverage{Type: "site"}); MapError(e).Code != "SITE_NOT_FOUND" {
		t.Fatal(e)
	}
}

func TestStage2AssociationDoesNotDuplicateObjects(t *testing.T) {
	ctx, s, _, site := disposableStore(t)
	d := discoveryFixture()
	if _, e := s.Discover(ctx, site, uuid.NewString(), d); e != nil {
		t.Fatal(e)
	}
	d.ExpectedStructureRevision = ptr(int64(1))
	d.BrowserIdentifier = "second-instance"
	if _, e := s.Discover(ctx, site, uuid.NewString(), d); e != nil {
		t.Fatal(e)
	}
	a, e := s.Assess(ctx, site, Coverage{Type: "page", Page: d.Page.URL})
	if e != nil || len(a.Communications) != 1 || a.Communications[0].Registration.Key != d.RegistrationKey || a.Page == nil || *a.Page != d.Page {
		t.Fatal(a, e)
	}
}
