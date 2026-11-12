package communications

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool        *pgxpool.Pool
	zones       map[string]bool
	hasher      PayloadHasher
	snapshotTTL time.Duration
}

func NewPostgresStore(pool *pgxpool.Pool, zones map[string]bool) *PostgresStore {
	copy := map[string]bool{}
	for z, exists := range zones {
		copy[z] = exists
	}
	return &PostgresStore{pool: pool, zones: copy}
}
func (s *PostgresStore) Find(ctx context.Context, site, key string) (Configuration, error) {
	parsed, err := uuid.Parse(key)
	if err != nil || parsed == uuid.Nil {
		return Configuration{}, ValidationErrors{"registration_key": {"must be a nonzero UUID"}}
	}
	key = parsed.String()
	tx, e := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return Configuration{}, e
	}
	defer tx.Rollback(ctx)
	result, e := s.findTx(ctx, tx, site, key, "")
	if e != nil {
		return result, readNotFound(ctx, tx, site, e)
	}
	return result, tx.Commit(ctx)
}

// readNotFound keeps an unknown site distinct from an unknown communication of a
// confirmed site, so a mistyped site is never reported as an unregistered object.
func readNotFound(ctx context.Context, tx pgx.Tx, site string, e error) error {
	if e != pgx.ErrNoRows {
		return e
	}
	var enrolled bool
	if failure := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public.sites WHERE site_key=$1)`, site).Scan(&enrolled); failure != nil {
		return failure
	}
	if !enrolled {
		return APIError{404, "site is not enrolled", "SITE_NOT_FOUND", nil}
	}
	return e
}
func (s *PostgresStore) findTx(ctx context.Context, tx pgx.Tx, site, key, page string) (Configuration, error) {
	return s.findEvidenceTx(ctx, tx, site, key, page, false)
}
func (s *PostgresStore) findEvidenceTx(ctx context.Context, tx pgx.Tx, site, key, page string, authoritative bool) (Configuration, error) {
	result := Configuration{History: []Event{}, Observations: []ObservedAssessment{}}
	var id int64
	var base string
	var approved, draft []byte
	e := tx.QueryRow(ctx, `SELECT o.id,s.base_url,o.registration_key::text,o.scope,o.component_identifier,p.url,
 o.structure_state,o.structure_revision,o.structure_issues,o.approval_status,
 CASE WHEN r.id IS NULL THEN NULL ELSE jsonb_build_object('revision',r.revision,'document',r.configuration,'structure_revision',r.structure_revision,'approved_by',r.decided_by,'approved_at',r.decided_at) END,
 CASE WHEN d.communication_object_id IS NULL THEN NULL ELSE jsonb_build_object('version',d.draft_version,'document',d.configuration,'based_on_revision',d.based_on_revision,'updated_by',d.updated_by,'updated_at',d.updated_at) END
 FROM public.communication_objects o JOIN public.sites s ON s.id=o.site_id
 LEFT JOIN public.pages p ON p.id=o.page_id AND p.site_id=o.site_id
 LEFT JOIN public.communication_configuration_revisions r ON r.id=o.approved_revision_id AND r.communication_object_id=o.id
 LEFT JOIN public.communication_configuration_drafts d ON d.communication_object_id=o.id AND d.site_id=o.site_id
 WHERE s.site_key=$1 AND o.registration_key=$2::uuid`, site, key).Scan(&id, &base, &result.Registration.Key, &result.Registration.Scope, &result.Registration.Component, &result.Registration.Page, &result.Structure.State, &result.Structure.Revision, &result.Structure.Issues, &result.ApprovalStatus, &approved, &draft)
	if e != nil {
		return result, e
	}
	if approved != nil {
		if e = json.Unmarshal(approved, &result.Approved); e != nil {
			return result, e
		}
		if e = s.validateEmbeddedDocument(approved); e != nil {
			return result, fmt.Errorf("stored approved configuration invalid")
		}
	}
	if draft != nil {
		if e = json.Unmarshal(draft, &result.Draft); e != nil {
			return result, e
		}
		if e = s.validateEmbeddedDocument(draft); e != nil {
			return result, fmt.Errorf("stored draft configuration invalid")
		}
	}
	result.History, e = eventsTx(ctx, tx, id)
	if e != nil {
		return result, e
	}

	// Finalized evidence is authoritative. Display may supplement uncovered pages
	// with their newest persisted browser instances, explicitly outside assessments.
	query := `SELECT v.id,p.url,v.canonical_destination,v.observed_method,v.observed_type,v.observed_zone FROM public.communication_current_observations v JOIN public.pages p ON p.id=v.page_id AND p.site_id=v.site_id WHERE v.communication_object_id=$1 AND ($2::text='' OR p.url=$2) ORDER BY v.id`
	if !authoritative {
		query = `WITH fallback AS (
   SELECT DISTINCT ON (v.page_id,v.browser_object_identifier) v.id,v.site_id,v.page_id,v.browser_object_identifier,v.canonical_destination,v.observed_method,v.observed_type,v.observed_zone
   FROM public.communication_observations v WHERE v.communication_object_id=$1 AND NOT EXISTS(SELECT 1 FROM public.communication_page_coverage c WHERE c.page_id=v.page_id AND c.current)
   ORDER BY v.page_id,v.browser_object_identifier,v.last_seen_at DESC,v.id DESC
  ), evidence AS (
   SELECT v.id,v.site_id,v.page_id,v.browser_object_identifier,v.canonical_destination,v.observed_method,v.observed_type,v.observed_zone FROM public.communication_current_observations v WHERE v.communication_object_id=$1
   UNION ALL SELECT * FROM fallback
  ) SELECT v.id,p.url,v.canonical_destination,v.observed_method,v.observed_type,v.observed_zone
  FROM evidence v JOIN public.pages p ON p.id=v.page_id AND p.site_id=v.site_id
  WHERE ($2::text='' OR p.url=$2) ORDER BY v.page_id,v.browser_object_identifier,v.id`
	}
	rows, e := tx.Query(ctx, query, id, page)
	if e != nil {
		return result, e
	}
	var doc *Document
	var bound int64
	if result.Approved != nil {
		doc = &result.Approved.Document
		bound = result.Approved.StructureRevision
	}
	result.Effective = "unapproved"
	for rows.Next() {
		var observed Observation
		if e = rows.Scan(&observed.ID, &observed.Page, &observed.Destination, &observed.Method, &observed.Type, &observed.Zone); e != nil {
			rows.Close()
			return result, e
		}
		evaluation := Evaluate(result.ApprovalStatus, result.Structure.State, result.Structure.Revision, bound, doc, observed, base)
		result.Observations = append(result.Observations, ObservedAssessment{observed, evaluation})
		result.Effective = combineEffective(result.Effective, evaluation.Effective)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return result, e
	}
	if len(result.Observations) == 0 && !authoritative {
		// An administrative definition without observations can still be reviewed;
		// assessments separately mark missing observation coverage as incomplete.
		var observed Observation
		e = tx.QueryRow(ctx, `SELECT detected_url,detected_method,object_type,detected_zone FROM public.communication_objects WHERE id=$1`, id).Scan(&observed.Destination, &observed.Method, &observed.Type, &observed.Zone)
		if e != nil {
			return result, e
		}
		result.Effective = Evaluate(result.ApprovalStatus, result.Structure.State, result.Structure.Revision, bound, doc, observed, base).Effective
	}
	return result, nil
}
func combineEffective(current, next string) string {
	rank := map[string]int{"approved": 0, "unapproved": 1, "rejected": 2, "reapproval_required": 3, "unauthorized": 4, "suspended": 5, "disabled": 6, "revoked": 7}
	// unapproved is the initializer; approved observations should replace it.
	if current == "" || current == "unapproved" && next == "approved" {
		return next
	}
	if rank[next] > rank[current] {
		return next
	}
	return current
}
func eventsTx(ctx context.Context, tx pgx.Tx, id int64) ([]Event, error) {
	events := []Event{}
	rows, e := tx.Query(ctx, `SELECT e.decision,e.actor,e.decided_at,r.revision,e.previous_status,e.resulting_status,e.structure_revision,e.reason FROM public.communication_approval_events e LEFT JOIN public.communication_configuration_revisions r ON r.id=e.configuration_revision_id WHERE e.communication_object_id=$1 ORDER BY e.decided_at,e.id`, id)
	if e != nil {
		return events, e
	}
	defer rows.Close()
	for rows.Next() {
		var event Event
		if e = rows.Scan(&event.Decision, &event.Actor, &event.DecidedAt, &event.Revision, &event.PreviousStatus, &event.ResultingStatus, &event.StructureRevision, &event.Reason); e != nil {
			return events, e
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
func (s *PostgresStore) History(ctx context.Context, site, key string) (History, error) {
	parsed, err := uuid.Parse(key)
	if err != nil || parsed == uuid.Nil {
		return History{}, ValidationErrors{"registration_key": {"must be a nonzero UUID"}}
	}
	key = parsed.String()
	result := History{History: []Event{}, Revisions: []Revision{}}
	tx, e := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return result, e
	}
	defer tx.Rollback(ctx)
	var id int64
	if e = tx.QueryRow(ctx, `SELECT o.id FROM public.communication_objects o JOIN public.sites s ON s.id=o.site_id WHERE s.site_key=$1 AND o.registration_key=$2::uuid`, site, key).Scan(&id); e != nil {
		return result, readNotFound(ctx, tx, site, e)
	}
	result.History, e = eventsTx(ctx, tx, id)
	if e != nil {
		return result, e
	}
	rows, e := tx.Query(ctx, `SELECT revision,configuration,status,structure_revision FROM public.communication_configuration_revisions WHERE communication_object_id=$1 ORDER BY revision`, id)
	if e != nil {
		return result, e
	}
	for rows.Next() {
		var r Revision
		var raw []byte
		if e = rows.Scan(&r.Revision, &raw, &r.Status, &r.StructureRevision); e != nil {
			rows.Close()
			return result, e
		}
		r.Document, e = Validate(raw, s.zones)
		if e != nil {
			rows.Close()
			return result, fmt.Errorf("stored revision invalid")
		}
		result.Revisions = append(result.Revisions, r)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return result, e
	}
	return result, tx.Commit(ctx)
}
func (s *PostgresStore) Decide(ctx context.Context, d Decision) (json.RawMessage, error) {
	if e := validateDecision(d); e != nil {
		return nil, e
	}

	key, _ := uuid.Parse(d.Key)
	d.Key = key.String()
	replayKey, _ := uuid.Parse(d.IdempotencyKey)
	d.IdempotencyKey = replayKey.String()
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	// Take the same object lock as the SECURITY DEFINER functions, so Go draft
	// validation and the function's decision cannot see different versions.
	var id int64
	if e = tx.QueryRow(ctx, `SELECT o.id FROM public.communication_objects o JOIN public.sites s ON s.id=o.site_id WHERE s.site_key=$1 AND o.registration_key=$2::uuid FOR UPDATE OF o`, d.Site, d.Key).Scan(&id); e != nil {
		return nil, e
	}
	var replay bool
	e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public.communication_approval_events WHERE idempotency_key=$1::uuid)`, d.IdempotencyKey).Scan(&replay)
	if e != nil {
		return nil, e
	}
	var q string
	var args []any
	switch d.Action {
	case "draft":
		doc, e := Validate(d.Document, s.zones)
		if e != nil {
			return nil, e
		}
		raw, _ := json.Marshal(doc)
		q = `SELECT public.codex_communications_save_draft($1::text,$2::uuid,$3::jsonb,$4::bigint,$5::text,$6::uuid)`
		args = []any{d.Site, d.Key, string(raw), *d.ExpectedVersion, d.Actor, d.IdempotencyKey}
	case "approve":
		if !replay {
			var raw []byte
			e = tx.QueryRow(ctx, `SELECT configuration FROM public.communication_configuration_drafts WHERE communication_object_id=$1`, id).Scan(&raw)
			if e == pgx.ErrNoRows {
				return nil, conflict("STALE_REVISION", "no draft to approve")
			}
			if e != nil {
				return nil, e
			}
			if _, e = Validate(raw, s.zones); e != nil {
				return nil, e
			}
		}
		q = `SELECT public.codex_communications_approve($1::text,$2::uuid,$3::bigint,$4::bigint,$5::text,$6::uuid,$7::text)`
		args = []any{d.Site, d.Key, *d.DraftVersion, *d.StructureRevision, d.Actor, d.IdempotencyKey, d.Reason}
	case "reject":
		q = `SELECT public.codex_communications_reject($1::text,$2::uuid,$3::bigint,$4::text,$5::uuid,$6::text)`
		args = []any{d.Site, d.Key, *d.DraftVersion, d.Actor, d.IdempotencyKey, d.Reason}
	case "disable":
		q = `SELECT public.codex_communications_disable($1::text,$2::uuid,$3::text,$4::uuid,$5::text)`
		args = []any{d.Site, d.Key, d.Actor, d.IdempotencyKey, d.Reason}
	case "revoke":
		q = `SELECT public.codex_communications_revoke($1::text,$2::uuid,$3::text,$4::uuid,$5::text)`
		args = []any{d.Site, d.Key, d.Actor, d.IdempotencyKey, d.Reason}
	}
	var result json.RawMessage
	if e = tx.QueryRow(ctx, q, args...).Scan(&result); e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	return result, nil
}
func validateDecision(d Decision) error {
	if id, e := uuid.Parse(d.Key); e != nil || id == uuid.Nil {
		return ValidationErrors{"registration_key": {"must be a nonzero UUID"}}
	}
	if id, e := uuid.Parse(d.IdempotencyKey); e != nil || id == uuid.Nil {
		return ValidationErrors{"idempotency_key": {"must be a nonzero UUID"}}
	}
	if d.Actor == "" || strings.TrimSpace(d.Actor) != d.Actor || len([]rune(d.Actor)) > 128 {
		return ValidationErrors{"actor": {"administrator identity required"}}
	}
	if d.Reason != nil && len([]rune(*d.Reason)) > 2000 {
		return ValidationErrors{"reason": {"must be at most 2000 characters"}}
	}
	switch d.Action {
	case "draft":
		if d.ExpectedVersion == nil || *d.ExpectedVersion < 0 {
			return ValidationErrors{"expected_version": {"must be a nonnegative integer"}}
		}
	case "approve":
		if d.StructureRevision == nil || *d.StructureRevision < 0 {
			return ValidationErrors{"structure_revision": {"must be a nonnegative integer"}}
		}
		fallthrough
	case "reject":
		if d.DraftVersion == nil || *d.DraftVersion < 1 {
			return ValidationErrors{"draft_version": {"must be a positive integer"}}
		}
	case "disable", "revoke":
	default:
		return ValidationErrors{"action": {"unsupported decision"}}
	}
	return nil
}

func (s *PostgresStore) validateEmbeddedDocument(raw []byte) error {
	var envelope struct {
		Document json.RawMessage `json:"document"`
	}
	if e := json.Unmarshal(raw, &envelope); e != nil {
		return e
	}
	_, e := Validate(envelope.Document, s.zones)
	return e
}
