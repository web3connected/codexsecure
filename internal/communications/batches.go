package communications

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type DiscoveryStart struct {
	Scope          string      `json:"scope"`
	Page           *SourcePage `json:"page"`
	ParentBatchKey *string     `json:"parent_batch_key"`
	ExpectedPages  []string    `json:"expected_pages"`
}
type DiscoveryFailure struct {
	Sequence        int     `json:"sequence"`
	Outcome         string  `json:"outcome"`
	Code            string  `json:"code"`
	RegistrationKey *string `json:"registration_key"`
}
type DiscoveryFinalize struct {
	Outcome              string `json:"outcome"`
	ExpectedObservations *int   `json:"expected_observations"`
}

// BatchStore keeps the lifecycle capability separate from configuration stores.
type BatchStore interface {
	StartDiscovery(context.Context, string, string, string, DiscoveryStart) (json.RawMessage, error)
	RecordDiscoveryFailure(context.Context, string, string, DiscoveryFailure) (json.RawMessage, error)
	FinalizeDiscovery(context.Context, string, string, string, DiscoveryFinalize) (json.RawMessage, error)
}

// DiscoveryFailureError preserves the ingestion error and exposes recording failure
// to server diagnostics without claiming that the ledger write succeeded.
type DiscoveryFailureError struct{ Original, Recording error }

func (e DiscoveryFailureError) Error() string {
	return fmt.Sprintf("%v (discovery failure ledger write failed: %v)", e.Original, e.Recording)
}
func (e DiscoveryFailureError) Unwrap() error { return e.Original }
func batchKey(key string) (string, error) {
	id, e := uuid.Parse(key)
	if e != nil || id == uuid.Nil {
		return "", ValidationErrors{"batch_key": {"must be a nonzero UUID"}}
	}
	return id.String(), nil
}
func (s *PostgresStore) batchCall(ctx context.Context, q string, args ...any) (json.RawMessage, error) {
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	var result json.RawMessage
	if e = tx.QueryRow(ctx, q, args...).Scan(&result); e != nil {
		return nil, e
	}
	return result, tx.Commit(ctx)
}
func (s *PostgresStore) StartDiscovery(ctx context.Context, site, key, actor string, d DiscoveryStart) (json.RawMessage, error) {
	key, e := batchKey(key)
	if e != nil {
		return nil, e
	}
	if strings.TrimSpace(actor) == "" || len([]rune(actor)) > 128 {
		return nil, ValidationErrors{"actor": {"verified service identity required"}}
	}
	var pageKey, pageURL, pageZone *string
	if d.Scope == "page" {
		if d.Page == nil || d.ExpectedPages != nil || d.Page.Key == "" || len([]rune(d.Page.Key)) > 128 || strings.TrimSpace(d.Page.Key) != d.Page.Key || !absoluteDestination(d.Page.URL) || !s.zones[d.Page.Zone] {
			return nil, ValidationErrors{"page": {"requires source page identity and no manifest"}}
		}
		pageKey, pageURL, pageZone = &d.Page.Key, &d.Page.URL, &d.Page.Zone
	} else if d.Scope != "site" || d.Page != nil || d.ParentBatchKey != nil || len(d.ExpectedPages) == 0 {
		return nil, ValidationErrors{"scope": {"requires page scope or site scope with a nonempty manifest"}}
	}
	for _, page := range d.ExpectedPages {
		if page == "" || len([]rune(page)) > 128 || strings.TrimSpace(page) != page {
			return nil, ValidationErrors{"expected_pages": {"must contain valid page keys"}}
		}
	}
	if d.ParentBatchKey != nil {
		v, e := batchKey(*d.ParentBatchKey)
		if e != nil {
			return nil, e
		}
		d.ParentBatchKey = &v
	}
	// Validate origin using the enrolled site before the function can create a page.
	if d.Page != nil {
		var base string
		if e = s.pool.QueryRow(ctx, `SELECT base_url FROM public.sites WHERE site_key=$1`, site).Scan(&base); e != nil {
			if e == pgx.ErrNoRows {
				return nil, APIError{404, "discovery batch not found", "DISCOVERY_BATCH_NOT_FOUND", nil}
			}
			return nil, e
		}
		if !sameOrigin(base, d.Page.URL) {
			return nil, ValidationErrors{"page.url": {"must belong to the registered site origin"}}
		}
	}
	return s.batchCall(ctx, `SELECT public.codex_communications_start_discovery($1::text,$2::uuid,$3::text,$4::text,$5::text,$6::text,$7::uuid,$8::text[],$9::text)`, site, key, d.Scope, pageKey, pageURL, pageZone, d.ParentBatchKey, d.ExpectedPages, actor)
}
func (s *PostgresStore) RecordDiscoveryFailure(ctx context.Context, site, key string, d DiscoveryFailure) (json.RawMessage, error) {
	key, e := batchKey(key)
	if e != nil {
		return nil, e
	}
	if d.Sequence < 1 || d.Sequence > 2147483647 || (d.Outcome != "rejected" && d.Outcome != "failed") {
		return nil, ValidationErrors{"failure": {"requires positive sequence and rejected or failed outcome"}}
	}
	if d.RegistrationKey != nil {
		v, e := batchKey(*d.RegistrationKey)
		if e != nil {
			return nil, e
		}
		d.RegistrationKey = &v
	}
	return s.batchCall(ctx, `SELECT public.codex_communications_record_discovery_failure($1::text,$2::uuid,$3::integer,$4::text,$5::text,$6::uuid)`, site, key, d.Sequence, d.Outcome, d.Code, d.RegistrationKey)
}
func (s *PostgresStore) FinalizeDiscovery(ctx context.Context, site, key, actor string, d DiscoveryFinalize) (json.RawMessage, error) {
	key, e := batchKey(key)
	if e != nil {
		return nil, e
	}
	if (d.Outcome != "complete" && d.Outcome != "failed") || d.ExpectedObservations != nil && (*d.ExpectedObservations < 0 || *d.ExpectedObservations > 2147483647) || strings.TrimSpace(actor) == "" || len([]rune(actor)) > 128 {
		return nil, ValidationErrors{"finalize": {"invalid outcome, count or verified actor"}}
	}
	return s.batchCall(ctx, `SELECT public.codex_communications_finalize_discovery($1::text,$2::uuid,$3::integer,$4::text,$5::text)`, site, key, d.ExpectedObservations, d.Outcome, actor)
}
