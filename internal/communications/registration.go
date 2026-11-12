package communications

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RegistrationRequest is an administrator's definition of one Link or Form. The
// destination and allowed methods live only in Document, so they cannot disagree
// with the configuration that is saved as the initial draft.
type RegistrationRequest struct {
	Type        string          `json:"type"`
	Scope       string          `json:"scope"`
	Component   *string         `json:"component"`
	Page        *SourcePage     `json:"page"`
	Declaration Declaration     `json:"declaration"`
	Document    json.RawMessage `json:"document"`
}
type RegistrationResult struct {
	Configuration
	RegistrationState string `json:"registration_state"`
	Replayed          bool   `json:"replayed,omitempty"`
}

// RegistrationStore keeps the capability separate from configuration stores, like BatchStore.
type RegistrationStore interface {
	Register(context.Context, string, string, string, string, RegistrationRequest) (RegistrationResult, error)
}

func (s *PostgresStore) validateRegistration(key, actor, replay string, r RegistrationRequest) (Document, error) {
	var doc Document
	expected := int64(0)
	if e := validateDecision(Decision{Action: "draft", Key: key, Actor: actor, IdempotencyKey: replay, ExpectedVersion: &expected}); e != nil {
		return doc, e
	}
	// Nav parents and fetch objects are not registrable through this contract.
	if r.Type != "link" && r.Type != "form" {
		return doc, ValidationErrors{"type": {"must be link or form"}}
	}
	if r.Scope != "universal" && r.Scope != "page" {
		return doc, ValidationErrors{"scope": {"must be universal or page"}}
	}
	if r.Component != nil && !validIdentifier(*r.Component) {
		return doc, ValidationErrors{"component": {"invalid registered component identifier"}}
	}
	if r.Scope == "page" {
		if r.Page == nil || r.Page.Key == "" || len([]rune(r.Page.Key)) > 128 || strings.TrimSpace(r.Page.Key) != r.Page.Key || !absoluteDestination(r.Page.URL) || !s.zones[r.Page.Zone] {
			return doc, ValidationErrors{"page": {"requires key, absolute HTTP(S) URL and registry zone"}}
		}
	} else if r.Page != nil {
		return doc, ValidationErrors{"page": {"universal communications have no owning page"}}
	}
	// The same checks discovery applies, so a valid registration is not invalidated
	// by the first matching observation.
	if issues := (Discovery{Scope: r.Scope, Component: r.Component, Declaration: r.Declaration}).issues(); len(issues) > 0 {
		return doc, ValidationErrors{"declaration": issues}
	}
	if *r.Declaration.Item != r.Type {
		return doc, ValidationErrors{"declaration.item": {"must match the communication type"}}
	}
	if r.Declaration.ItemIdentifier == nil {
		return doc, ValidationErrors{"declaration.item_identifier": {"is required"}}
	}
	doc, e := Validate(r.Document, s.zones)
	if e != nil {
		return doc, e
	}
	if doc.Type != r.Type {
		return doc, ValidationErrors{"document.type": {"must match the communication type"}}
	}
	return doc, nil
}

// registeredDestination resolves a configured destination to the absolute baseline
// the registry stores; relative paths resolve against the site's origin, as in Evaluate.
func registeredDestination(destination, base string) (string, error) {
	u, e := url.Parse(destination)
	if e == nil && !u.IsAbs() {
		var b *url.URL
		if b, e = url.Parse(base); e == nil {
			b.Path = "/"
			b.RawPath = ""
			b.RawQuery = ""
			b.Fragment = ""
			u = b.ResolveReference(u)
		}
	}
	if e != nil || !absoluteDestination(u.String()) {
		return "", ValidationErrors{"document.destination": {"must resolve to an absolute HTTP(S) URL"}}
	}
	return u.String(), nil
}

// Register creates one unapproved communication and its initial draft in one
// transaction. The UUID stays the caller's resolved identity, as in Discover. Only
// the existing draft function writes configuration, and nothing here can approve.
func (s *PostgresStore) Register(ctx context.Context, site, key, actor, replay string, r RegistrationRequest) (result RegistrationResult, err error) {
	doc, e := s.validateRegistration(key, actor, replay, r)
	if e != nil {
		return result, e
	}
	parsed, _ := uuid.Parse(key)
	key = parsed.String()
	parsed, _ = uuid.Parse(replay)
	replay = parsed.String()
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return result, e
	}
	defer tx.Rollback(ctx)
	var siteID int64
	var base string
	e = tx.QueryRow(ctx, `SELECT id,base_url FROM public.sites WHERE site_key=$1`, site).Scan(&siteID, &base)
	if e == pgx.ErrNoRows {
		return result, APIError{404, "site is not enrolled", "SITE_NOT_FOUND", nil}
	}
	if e != nil {
		return result, e
	}
	// The discovery lock also serializes creating a registry row for this site.
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('codexsecure.discovery:' || $1::text,0))`, site); e != nil {
		return result, e
	}
	var ownerPage *int64
	pageURL := ""
	if r.Page != nil {
		pageURL = r.Page.URL
		if !sameOrigin(pageURL, base) {
			return result, ValidationErrors{"page.url": {"source page must belong to the registered site origin"}}
		}
		var pageID int64
		var existingURL, existingZone string
		e = tx.QueryRow(ctx, `SELECT id,url,zone FROM public.pages WHERE site_id=$1 AND page_key=$2`, siteID, r.Page.Key).Scan(&pageID, &existingURL, &existingZone)
		if e == pgx.ErrNoRows {
			e = tx.QueryRow(ctx, `INSERT INTO public.pages(site_id,page_key,url,zone) VALUES($1,$2,$3,$4) RETURNING id`, siteID, r.Page.Key, r.Page.URL, r.Page.Zone).Scan(&pageID)
		} else if e == nil && (existingURL != r.Page.URL || existingZone != r.Page.Zone) {
			return result, conflict("REGISTRATION_IDENTITY_CONFLICT", "source page identity is immutable")
		}
		if e != nil {
			return result, e
		}
		ownerPage = &pageID
	}
	destination, e := registeredDestination(doc.Destination, base)
	if e != nil {
		return result, e
	}
	var objectID int64
	var objectPage *int64
	var scope, kind string
	var component *string
	e = tx.QueryRow(ctx, `SELECT id,page_id,scope,component_identifier,object_type FROM public.communication_objects WHERE site_id=$1 AND registration_key=$2::uuid FOR UPDATE`, siteID, key).Scan(&objectID, &objectPage, &scope, &component, &kind)
	switch {
	case e == pgx.ErrNoRows:
		// Same declaration ownership lookup as Discover: an item observed under one UUID cannot gain another.
		var registered string
		e = tx.QueryRow(ctx, `SELECT DISTINCT o.registration_key::text FROM public.communication_objects o
   JOIN public.communication_observations v ON v.communication_object_id=o.id AND v.site_id=o.site_id
   JOIN public.pages p ON p.id=v.page_id AND p.site_id=v.site_id
   WHERE o.site_id=$1 AND o.scope=$2 AND o.component_identifier IS NOT DISTINCT FROM $3::varchar
   AND v.metadata->'declaration'->>'item_identifier'=$4
   AND ($2='universal' OR p.url=$5) LIMIT 1`, siteID, r.Scope, r.Component, *r.Declaration.ItemIdentifier, pageURL).Scan(&registered)
		if e == nil {
			return result, conflict("REGISTRATION_IDENTITY_CONFLICT", "declaration already belongs to registered UUID "+registered)
		}
		if e != pgx.ErrNoRows {
			return result, e
		}
		e = tx.QueryRow(ctx, `INSERT INTO public.communication_objects(site_id,page_id,registration_key,object_type,detected_url,detected_method,detected_zone,canonical_destination,scope,component_identifier) VALUES($1,$2,$3::uuid,$4,$5,$6,$7,$5,$8,$9) RETURNING id`, siteID, ownerPage, key, r.Type, destination, doc.Methods[0], doc.Zone, r.Scope, r.Component).Scan(&objectID)
		if e != nil {
			return result, e
		}
		if _, e = tx.Exec(ctx, `UPDATE public.communication_objects SET structure_state='valid',structure_issues='[]'::jsonb,structure_revision=1 WHERE id=$1`, objectID); e != nil {
			return result, e
		}
	case e != nil:
		return result, e
	default:
		if scope != r.Scope || kind != r.Type || !sameOptional(component, r.Component) || (scope == "page" && (objectPage == nil || ownerPage == nil || *objectPage != *ownerPage)) {
			return result, conflict("REGISTRATION_IDENTITY_CONFLICT", "registered type, scope, component and page ownership cannot change")
		}
		// Only this request's own retry reaches the draft function, which then verifies
		// the recorded actor and document digest instead of saving another draft.
		if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public.communication_approval_events WHERE communication_object_id=$1 AND idempotency_key=$2::uuid AND decision='draft_saved')`, objectID, replay).Scan(&result.Replayed); e != nil {
			return result, e
		}
		if !result.Replayed {
			return result, conflict("ALREADY_REGISTERED", "communication is already registered; use its configuration operations")
		}
	}
	raw, _ := json.Marshal(doc)
	var saved json.RawMessage
	if e = tx.QueryRow(ctx, `SELECT public.codex_communications_save_draft($1::text,$2::uuid,$3::jsonb,$4::bigint,$5::text,$6::uuid)`, site, key, string(raw), int64(0), actor, replay).Scan(&saved); e != nil {
		return result, e
	}
	if result.Configuration, e = s.findTx(ctx, tx, site, key, ""); e != nil {
		return result, e
	}
	result.RegistrationState = result.ApprovalStatus
	if result.ApprovalStatus == "unapproved" {
		result.RegistrationState = "pending_approval"
	}
	return result, tx.Commit(ctx)
}

var _ RegistrationStore = (*PostgresStore)(nil)
