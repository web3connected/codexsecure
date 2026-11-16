package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Harmonic Alignment relationship recording (CodexSecure Step 3B; CIR-2026-11-14,
// HA-2026-11-14, S1-1 to S1-4). One aligned Z1 GET observation of a site communication
// relationship identified by its PageMeta package UUID. Communication relationships
// only: nothing about the visitor is accepted or stored.
//
// Only public.pagemeta_packages (SELECT) and public.harmonic_zones (INSERT, UPDATE) are
// touched, as codex_zones_runtime. Weights and misalignment_count are never written.

// AlignmentAligned is the only outcome Step 3B records.
const AlignmentAligned = "aligned"

var (
	// ErrAlignmentStoreUnavailable: no database connection.
	ErrAlignmentStoreUnavailable = errors.New("alignment store unavailable")
	// ErrAlignmentPageMetaNotFound: no PageMeta package has the UUID.
	ErrAlignmentPageMetaNotFound = errors.New("pagemeta package not found")
	// ErrAlignmentEvidenceMismatch: the CodexHash v2 evidence is not the package's.
	ErrAlignmentEvidenceMismatch = errors.New("codexhash evidence does not match the pagemeta package")
	// ErrAlignmentRelationshipConflict: the stored relationship disagrees with the package.
	ErrAlignmentRelationshipConflict = errors.New("stored relationship disagrees with the pagemeta package")
)

// AlignmentMismatch reports the first relationship field that disagrees with PageMeta.
type AlignmentMismatch struct{ Field string }

func (e *AlignmentMismatch) Error() string {
	return fmt.Sprintf("%s does not match the pagemeta package", e.Field)
}

// AlignmentObservation is one observed communication of a relationship.
type AlignmentObservation struct {
	PageMetaUUID string
	Zone         string
	RouteName    *string
	HTTPMethod   string
	CodexHash    string
	TIU          float64
	Intent       *string
	Controller   *string
	Action       *string
}

// AlignmentRecord is the relationship after the observation.
type AlignmentRecord struct {
	Result            string
	Created           bool
	ID                int64
	PageMetaUUID      string
	Zone              string
	RouteName         *string
	HTTPMethod        string
	AlignmentCount    int64
	MisalignmentCount int64
	LastAlignedAt     time.Time
	LastTIU           string
}

// RecordAlignment verifies the observation against the authoritative PageMeta package and
// records it as aligned, in one transaction. The relationship row is created on the first
// observation and its alignment_count incremented atomically afterwards (no read-modify-
// write), so concurrent observations neither duplicate the row nor lose increments.
func (db *DB) RecordAlignment(ctx context.Context, obs AlignmentObservation) (AlignmentRecord, error) {
	if db == nil || db.Pool == nil {
		return AlignmentRecord{}, ErrAlignmentStoreUnavailable
	}
	tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AlignmentRecord{}, fmt.Errorf("begin alignment transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	record, err := RecordAlignmentTx(ctx, tx, obs)
	if err != nil {
		return AlignmentRecord{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AlignmentRecord{}, fmt.Errorf("commit alignment transaction: %w", err)
	}
	return record, nil
}

// RecordAlignmentTx is RecordAlignment inside a caller's transaction.
func RecordAlignmentTx(ctx context.Context, tx pgx.Tx, obs AlignmentObservation) (AlignmentRecord, error) {
	var hash string
	var method, zone, routeName, controller, action, intent *string
	err := tx.QueryRow(ctx, `
		SELECT hash,
		       payload->'data'->>'method',
		       payload->'data'->>'declared_zone',
		       payload->'data'->'route'->>'name',
		       payload->'data'->'route'->>'controller',
		       payload->'data'->'route'->>'action',
		       payload->'data'->>'intent'
		FROM pagemeta_packages
		WHERE uuid = $1::uuid`, obs.PageMetaUUID).Scan(&hash, &method, &zone, &routeName, &controller, &action, &intent)
	if errors.Is(err, pgx.ErrNoRows) {
		return AlignmentRecord{}, ErrAlignmentPageMetaNotFound
	}
	if err != nil {
		return AlignmentRecord{}, fmt.Errorf("read pagemeta package: %w", err)
	}

	// The relationship must be the one PageMeta established: method, zone, route and, when
	// supplied, the developer intent; the evidence must be the package's CodexHash v2 hash.
	switch {
	case !equalText(method, &obs.HTTPMethod):
		return AlignmentRecord{}, &AlignmentMismatch{Field: "relationship.http_method"}
	case !equalText(zone, &obs.Zone):
		return AlignmentRecord{}, &AlignmentMismatch{Field: "relationship.zone"}
	case !equalText(routeName, obs.RouteName):
		return AlignmentRecord{}, &AlignmentMismatch{Field: "relationship.route_name"}
	case obs.Intent != nil && !equalText(intent, obs.Intent):
		return AlignmentRecord{}, &AlignmentMismatch{Field: "developer_intent.intent"}
	case obs.Controller != nil && !equalText(controller, obs.Controller):
		return AlignmentRecord{}, &AlignmentMismatch{Field: "developer_intent.controller"}
	case obs.Action != nil && !equalText(action, obs.Action):
		return AlignmentRecord{}, &AlignmentMismatch{Field: "developer_intent.action"}
	case hash != obs.CodexHash:
		return AlignmentRecord{}, ErrAlignmentEvidenceMismatch
	}

	record := AlignmentRecord{Result: AlignmentAligned}
	err = tx.QueryRow(ctx, `
		INSERT INTO harmonic_zones
		       (pagemeta_uuid, zone, route_name, http_method, alignment_count, misalignment_count,
		        last_aligned_at, last_tiu, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, $4, 1, 0, now(), $5, now(), now())
		ON CONFLICT (pagemeta_uuid) DO UPDATE
		SET alignment_count = harmonic_zones.alignment_count + 1,
		    last_aligned_at = GREATEST(harmonic_zones.last_aligned_at, EXCLUDED.last_aligned_at),
		    last_tiu        = GREATEST(harmonic_zones.last_tiu, EXCLUDED.last_tiu),
		    updated_at      = now()
		WHERE harmonic_zones.zone = EXCLUDED.zone
		  AND harmonic_zones.route_name IS NOT DISTINCT FROM EXCLUDED.route_name
		  AND harmonic_zones.http_method = EXCLUDED.http_method
		RETURNING id, pagemeta_uuid::text, zone, route_name, http_method, alignment_count,
		          misalignment_count, last_aligned_at, last_tiu::text, (xmax = 0)`,
		obs.PageMetaUUID, obs.Zone, obs.RouteName, obs.HTTPMethod, obs.TIU,
	).Scan(&record.ID, &record.PageMetaUUID, &record.Zone, &record.RouteName, &record.HTTPMethod,
		&record.AlignmentCount, &record.MisalignmentCount, &record.LastAlignedAt, &record.LastTIU, &record.Created)
	if errors.Is(err, pgx.ErrNoRows) {
		// ON CONFLICT … WHERE refused: the stored row describes another relationship.
		return AlignmentRecord{}, ErrAlignmentRelationshipConflict
	}
	if err != nil {
		return AlignmentRecord{}, fmt.Errorf("record alignment: %w", err)
	}
	return record, nil
}

func equalText(stored, supplied *string) bool {
	if stored == nil || supplied == nil {
		return stored == nil && supplied == nil
	}
	return *stored == *supplied
}
