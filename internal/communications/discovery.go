package communications

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SourcePage struct {
	Key  string `json:"key"`
	URL  string `json:"url"`
	Zone string `json:"zone"`
}
type Detected struct {
	Destination     string `json:"destination"`
	Method          string `json:"method"`
	TransportMethod string `json:"transport_method"`
	Type            string `json:"type"`
	Zone            string `json:"zone"`
}
type Declaration struct {
	Scope          *string `json:"scope"`
	Component      *string `json:"component"`
	Item           *string `json:"item"`
	ItemIdentifier *string `json:"item_identifier"`
}
type DiscoveryBatch struct {
	Key      string `json:"key"`
	Sequence int    `json:"sequence"`
}
type Discovery struct {
	Batch                     *DiscoveryBatch `json:"batch,omitempty"`
	RegistrationKey           string          `json:"registration_key"`
	Scope                     string          `json:"scope"`
	Component                 *string         `json:"component"`
	Page                      SourcePage      `json:"page"`
	Detected                  Detected        `json:"detected"`
	Declaration               Declaration     `json:"declaration"`
	BrowserIdentifier         string          `json:"browser_object_identifier"`
	OriginalURL               *string         `json:"original_url"`
	OriginalMethod            *string         `json:"original_method"`
	ExpectedStructureRevision *int64          `json:"expected_structure_revision"`
}
type DiscoveryResult struct {
	Registration        Registration `json:"registration"`
	Structure           Structure    `json:"structure"`
	ObservedInstanceKey string       `json:"observed_instance_key"`
	Replayed            bool         `json:"replayed,omitempty"`
}
type discoveryMetadata struct {
	RequestSHA256 string          `json:"request_sha256"`
	Declaration   Declaration     `json:"declaration"`
	Result        DiscoveryResult `json:"result"`
}

var identifierPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func validIdentifier(v string) bool  { return len(v) <= 64 && identifierPattern.MatchString(v) }
func sameOptional(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func (d Discovery) issues() []string {
	issues := []string{}
	if d.Declaration.Scope == nil {
		issues = append(issues, "missing_scope")
	} else if *d.Declaration.Scope != "page" && *d.Declaration.Scope != "universal" {
		issues = append(issues, "invalid_scope")
	} else if *d.Declaration.Scope != d.Scope {
		issues = append(issues, "scope_mismatch")
	}
	if d.Declaration.Component != nil && !validIdentifier(*d.Declaration.Component) {
		issues = append(issues, "invalid_component")
	} else if d.Scope == "universal" && d.Declaration.Component == nil {
		issues = append(issues, "missing_component")
	} else if !sameOptional(d.Component, d.Declaration.Component) {
		issues = append(issues, "component_mismatch")
	}
	if d.Scope == "universal" && d.Component == nil {
		issues = append(issues, "missing_registered_component")
	}
	if d.Declaration.Item == nil {
		issues = append(issues, "missing_item")
	} else {
		item := *d.Declaration.Item
		if item == "ajax" {
			item = "fetch"
		}
		if item != "link" && item != "form" && item != "fetch" {
			issues = append(issues, "invalid_item")
		}
	}
	if d.Declaration.ItemIdentifier != nil && !validIdentifier(*d.Declaration.ItemIdentifier) {
		issues = append(issues, "invalid_item_identifier")
	}
	return issues
}
func (s *PostgresStore) validateDiscovery(d *Discovery) error {
	if key, e := uuid.Parse(d.RegistrationKey); e != nil || key == uuid.Nil {
		return ValidationErrors{"registration_key": {"must be a registered nonzero UUID; declaration keys are not UUIDs"}}
	} else {
		d.RegistrationKey = key.String()
	}
	if d.ExpectedStructureRevision == nil || *d.ExpectedStructureRevision < 0 {
		return ValidationErrors{"expected_structure_revision": {"must be a nonnegative integer"}}
	}
	if d.Scope != "universal" && d.Scope != "page" {
		return ValidationErrors{"scope": {"must be universal or page"}}
	}
	if d.Component != nil && !validIdentifier(*d.Component) {
		return ValidationErrors{"component": {"invalid registered component identifier"}}
	}
	if d.Page.Key == "" || len([]rune(d.Page.Key)) > 128 || strings.TrimSpace(d.Page.Key) != d.Page.Key || !absoluteDestination(d.Page.URL) || !s.zones[d.Page.Zone] {
		return ValidationErrors{"page": {"requires key, absolute HTTP(S) URL and registry zone"}}
	}
	if d.BrowserIdentifier == "" || len(d.BrowserIdentifier) > 2048 || strings.TrimSpace(d.BrowserIdentifier) != d.BrowserIdentifier {
		return ValidationErrors{"browser_object_identifier": {"must be a nonempty identifier of at most 2048 bytes"}}
	}
	if d.Detected.Type == "ajax" {
		d.Detected.Type = "fetch"
	}
	// Observed behavior is evidence, not an administrator policy. In particular,
	// a recognized method must not be discarded for violating policy type rules.
	if d.Detected.Type != "link" && d.Detected.Type != "form" && d.Detected.Type != "fetch" {
		return ValidationErrors{"detected.type": {"must be link, form or fetch"}}
	}
	if !s.zones[d.Detected.Zone] {
		return ValidationErrors{"detected.zone": {"must exist in the zone registry"}}
	}
	if !absoluteDestination(d.Detected.Destination) {
		return ValidationErrors{"detected.destination": {"must be an absolute HTTP(S) URL"}}
	}
	d.Detected.Method = strings.ToUpper(strings.TrimSpace(d.Detected.Method))
	methodOK := false
	for _, method := range methodOrder {
		if d.Detected.Method == method {
			methodOK = true
		}
	}
	if !methodOK {
		return ValidationErrors{"detected.method": {"must be a supported HTTP method"}}
	}
	supported := false
	for _, m := range methodOrder {
		if d.Detected.TransportMethod == m {
			supported = true
		}
	}
	if !supported {
		return ValidationErrors{"detected.transport_method": {"must be a supported uppercase HTTP method"}}
	}

	return nil
}
func absoluteDestination(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.IsAbs() && ValidDestination(raw)
}
func sameOrigin(a, b string) bool {
	u, e := url.Parse(a)
	if e != nil {
		return false
	}
	v, e := url.Parse(b)
	return e == nil && strings.EqualFold(u.Scheme, v.Scheme) && strings.EqualFold(u.Host, v.Host)
}

// Discover accepts an existing registered UUID; it never derives a second identity
// from a route, destination, or declaration. Site enrollment remains administrative.
func (s *PostgresStore) Discover(ctx context.Context, site, replay string, d Discovery) (result DiscoveryResult, err error) {
	if d.Batch != nil {
		key, e := uuid.Parse(d.Batch.Key)
		if e != nil || key == uuid.Nil || d.Batch.Sequence < 1 || d.Batch.Sequence > 2147483647 {
			return result, ValidationErrors{"batch": {"requires a nonzero UUID and positive integer sequence"}}
		}
		batch := *d.Batch
		batch.Key = key.String()
		d.Batch = &batch
		// Registered before transaction cleanup, so rollback runs before this callback.
		defer func() {
			if err == nil {
				return
			}
			api := MapError(err)
			outcome := "rejected"
			if api.Status >= 500 {
				outcome = "failed"
			}
			failureCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			var registration *string
			if id, e := uuid.Parse(d.RegistrationKey); e == nil && id != uuid.Nil {
				v := id.String()
				registration = &v
			}
			_, recordingError := s.RecordDiscoveryFailure(failureCtx, site, batch.Key, DiscoveryFailure{Sequence: batch.Sequence, Outcome: outcome, Code: api.Code, RegistrationKey: registration})
			if recordingError != nil {
				err = DiscoveryFailureError{Original: err, Recording: recordingError}
			}
		}()
	}

	if e := s.validateDiscovery(&d); e != nil {
		return result, e
	}
	key, e := uuid.Parse(replay)
	if e != nil || key == uuid.Nil {
		return result, ValidationErrors{"idempotency_key": {"must be a nonzero UUID"}}
	}
	replay = key.String()
	raw, _ := json.Marshal(d)
	digest := sha256.Sum256(raw)
	fingerprint := hex.EncodeToString(digest[:])
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return result, e
	}
	defer tx.Rollback(ctx)
	// Serialize site registry ingestion, including creation and replay checks. This
	// lock supplements, and never replaces, database ownership/approval protections.
	var siteID int64
	var base string
	e = tx.QueryRow(ctx, `SELECT id,base_url FROM public.sites WHERE site_key=$1`, site).Scan(&siteID, &base)
	if e == pgx.ErrNoRows {
		return result, APIError{404, "site is not enrolled", "SITE_NOT_FOUND", nil}
	}
	if e != nil {
		return result, e
	}
	if !sameOrigin(d.Page.URL, base) {
		return result, ValidationErrors{"page.url": {"source page must belong to the registered site origin"}}
	}
	if _, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('codexsecure.discovery:' || $1::text,0))`, site); e != nil {
		return result, e
	}
	var prior []byte
	e = tx.QueryRow(ctx, `SELECT metadata FROM public.communication_observations WHERE site_id=$1 AND observed_instance_key=$2::uuid ORDER BY id LIMIT 1`, siteID, replay).Scan(&prior)
	if e == nil {
		var m discoveryMetadata
		if e = json.Unmarshal(prior, &m); e != nil {
			return result, e
		}
		if m.RequestSHA256 != fingerprint {
			return result, conflict("ALREADY_PROCESSED", "discovery replay key was used for another payload")
		}
		m.Result.Replayed = true
		return m.Result, tx.Commit(ctx)
	} else if e != pgx.ErrNoRows {
		return result, e
	}
	var pageID int64
	var pageURL, pageZone string
	e = tx.QueryRow(ctx, `SELECT id,url,zone FROM public.pages WHERE site_id=$1 AND page_key=$2`, siteID, d.Page.Key).Scan(&pageID, &pageURL, &pageZone)
	if e == pgx.ErrNoRows {
		e = tx.QueryRow(ctx, `INSERT INTO public.pages(site_id,page_key,url,zone) VALUES($1,$2,$3,$4) RETURNING id`, siteID, d.Page.Key, d.Page.URL, d.Page.Zone).Scan(&pageID)
	} else if e == nil && (pageURL != d.Page.URL || pageZone != d.Page.Zone) {
		return result, conflict("REGISTRATION_IDENTITY_CONFLICT", "source page identity is immutable")
	}
	if e != nil {
		return result, e
	}

	// Compare the PHP declaration ownership fields directly. UUID assignment is
	// separate; destination, method and DOM position never enter this lookup.
	// The site lock serializes this persistent declaration-to-UUID association.
	if d.Declaration.ItemIdentifier != nil && validIdentifier(*d.Declaration.ItemIdentifier) {
		var registered string
		err := tx.QueryRow(ctx, `SELECT DISTINCT o.registration_key::text FROM public.communication_objects o
   JOIN public.communication_observations v ON v.communication_object_id=o.id AND v.site_id=o.site_id
   JOIN public.pages p ON p.id=v.page_id AND p.site_id=v.site_id
   WHERE o.site_id=$1 AND o.scope=$2 AND o.component_identifier IS NOT DISTINCT FROM $3::varchar
   AND v.metadata->'declaration'->>'item_identifier'=$4
   AND ($2='universal' OR p.url=$5) LIMIT 1`, siteID, d.Scope, d.Component, *d.Declaration.ItemIdentifier, d.Page.URL).Scan(&registered)
		if err == nil && registered != d.RegistrationKey {
			return result, conflict("REGISTRATION_IDENTITY_CONFLICT", "declaration already belongs to registered UUID "+registered)
		}
		if err != nil && err != pgx.ErrNoRows {
			return result, err
		}
	}
	var objectID, revision int64
	var objectPage *int64
	var scope, state string
	var component *string
	var issues []string
	e = tx.QueryRow(ctx, `SELECT id,page_id,scope,component_identifier,structure_state,structure_revision,structure_issues FROM public.communication_objects WHERE site_id=$1 AND registration_key=$2::uuid FOR UPDATE`, siteID, d.RegistrationKey).Scan(&objectID, &objectPage, &scope, &component, &state, &revision, &issues)
	if e == pgx.ErrNoRows {
		if *d.ExpectedStructureRevision != 0 {
			return result, conflict("STALE_REVISION", "new registrations require expected structural revision 0")
		}
		var ownerPage *int64
		if d.Scope == "page" {
			ownerPage = &pageID
		}
		e = tx.QueryRow(ctx, `INSERT INTO public.communication_objects(site_id,page_id,registration_key,object_type,detected_url,detected_method,detected_zone,canonical_destination,scope,component_identifier) VALUES($1,$2,$3::uuid,$4,$5,$6,$7,$5,$8,$9) RETURNING id`, siteID, ownerPage, d.RegistrationKey, d.Detected.Type, d.Detected.Destination, d.Detected.Method, d.Detected.Zone, d.Scope, d.Component).Scan(&objectID)
		if e != nil {
			return result, e
		}
		revision = 0
		state = "unknown"
		issues = []string{}
	} else if e != nil {
		return result, e
	} else {
		if scope != d.Scope || !sameOptional(component, d.Component) || (scope == "page" && (objectPage == nil || *objectPage != pageID)) {
			return result, conflict("REGISTRATION_IDENTITY_CONFLICT", "registered scope, component and page ownership cannot change")
		}

		if d.Declaration.ItemIdentifier != nil && validIdentifier(*d.Declaration.ItemIdentifier) {
			var registeredItem *string
			err := tx.QueryRow(ctx, `SELECT metadata->'declaration'->>'item_identifier' FROM public.communication_observations WHERE communication_object_id=$1 AND (metadata->'declaration'->>'item_identifier') ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(metadata->'declaration'->>'item_identifier')<=64 ORDER BY id LIMIT 1`, objectID).Scan(&registeredItem)
			if err != nil && err != pgx.ErrNoRows {
				return result, err
			}
			if err == nil && registeredItem != nil && validIdentifier(*registeredItem) && *registeredItem != *d.Declaration.ItemIdentifier {
				return result, conflict("REGISTRATION_IDENTITY_CONFLICT", "registered item identifier cannot change")
			}
		}
		if revision != *d.ExpectedStructureRevision {
			return result, conflict("STALE_REVISION", "structural revision changed")
		}
	}
	newIssues := d.issues()
	newState := "valid"
	if len(newIssues) > 0 {
		newState = "invalid"
	}
	changed := state != newState || !equalStrings(issues, newIssues)
	// Declaration changes count even when the list of validation issues is equal.
	var oldMetadata []byte
	e = tx.QueryRow(ctx, `SELECT metadata FROM public.communication_observations WHERE communication_object_id=$1 ORDER BY id DESC LIMIT 1`, objectID).Scan(&oldMetadata)
	if e == nil {
		var old discoveryMetadata
		if e = json.Unmarshal(oldMetadata, &old); e != nil {
			return result, e
		}
		before, _ := json.Marshal(old.Declaration)
		after, _ := json.Marshal(d.Declaration)
		if string(before) != string(after) {
			changed = true
		}
	} else if e != pgx.ErrNoRows {
		return result, e
	}
	if changed {
		revision++
		issuesJSON, _ := json.Marshal(newIssues)
		if _, e = tx.Exec(ctx, `UPDATE public.communication_objects SET structure_state=$2,structure_issues=$3::jsonb,structure_revision=$4 WHERE id=$1`, objectID, newState, string(issuesJSON), revision); e != nil {
			return result, e
		}
	}
	result = DiscoveryResult{Registration: Registration{Key: d.RegistrationKey, Scope: d.Scope, Component: d.Component}, Structure: Structure{State: newState, Revision: revision, Issues: newIssues}, ObservedInstanceKey: replay}
	if d.Scope == "page" {
		result.Registration.Page = &d.Page.URL
	}
	metadata, _ := json.Marshal(discoveryMetadata{fingerprint, d.Declaration, result})
	var batchID *int64
	var sequence *int
	if d.Batch != nil {
		var id int64
		e = tx.QueryRow(ctx, `SELECT b.id FROM public.communication_discovery_batches b JOIN public.sites s ON s.id=b.site_id WHERE s.site_key=$1 AND b.batch_key=$2::uuid`, site, d.Batch.Key).Scan(&id)
		if e == pgx.ErrNoRows {
			return result, APIError{404, "discovery batch not found", "DISCOVERY_BATCH_NOT_FOUND", nil}
		}
		if e != nil {
			return result, e
		}
		batchID, sequence = &id, &d.Batch.Sequence
	}
	_, e = tx.Exec(ctx, `INSERT INTO public.communication_observations(site_id,page_id,communication_object_id,observed_instance_key,browser_object_identifier,original_url,original_method,observed_method,transport_method,observed_zone,canonical_destination,metadata,observed_type,discovery_batch_id,batch_sequence) VALUES($1,$2,$3,$4::uuid,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13,$14,$15)`, siteID, pageID, objectID, replay, d.BrowserIdentifier, d.OriginalURL, d.OriginalMethod, d.Detected.Method, d.Detected.TransportMethod, d.Detected.Zone, d.Detected.Destination, string(metadata), d.Detected.Type, batchID, sequence)
	if e != nil {
		return result, e
	}

	if _, e = tx.Exec(ctx, `UPDATE public.communication_objects SET last_seen_at=clock_timestamp() WHERE id=$1`, objectID); e != nil {
		return result, e
	}
	if _, e = tx.Exec(ctx, `UPDATE public.pages SET last_seen_at=clock_timestamp() WHERE id=$1`, pageID); e != nil {
		return result, e
	}
	return result, tx.Commit(ctx)
}
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

var _ Store = (*PostgresStore)(nil)
