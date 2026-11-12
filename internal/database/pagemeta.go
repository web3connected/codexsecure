package database

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/jackc/pgx/v5"
)

// PageMeta recognition results.
const (
	PageMetaRecorded             = "recorded"              // new page: package inserted
	PageMetaKnown                = "known"                 // same structural hash already recorded
	PageMetaConfigurationChanged = "configuration_changed" // known page, different hash: not recorded
)

// PageMetaRecord is a validated protected PageMeta package.
type PageMetaRecord struct {
	Method  string // payload.data.method — part of page identity
	URI     string // payload.data.route.uri — part of page identity
	Hash    string // verified structural CodexHash (SHA-512 of canonical data)
	TIU     string // CodexTime TIU as a decimal string (numeric(20,6))
	Payload []byte // complete protected package, stored as received (JSONB)
}

// PageMetaPackage is a row of pagemeta_packages (payload omitted).
type PageMetaPackage struct {
	UUID      string    `json:"uuid"`
	Hash      string    `json:"hash"`
	TIU       string    `json:"tiu"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PageMetaResult describes what recording did.
type PageMetaResult struct {
	Result        string            `json:"result"`
	Recorded      bool              `json:"recorded"`
	Package       *PageMetaPackage  `json:"package"`
	KnownPackages []PageMetaPackage `json:"known_packages"`
}

// ErrDatabaseUnavailable wraps database failures for callers that must not expose details.
var ErrDatabaseUnavailable = errors.New("codexzone database unavailable")

// RecordPageMeta recognizes or records a PageMeta package in one transaction.
// Only public.pagemeta_packages is read or written (SELECT, INSERT).
func (db *DB) RecordPageMeta(ctx context.Context, record PageMetaRecord) (PageMetaResult, error) {
	tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return PageMetaResult{}, fmt.Errorf("%w: begin: %v", ErrDatabaseUnavailable, err)
	}
	defer tx.Rollback(ctx)

	result, err := RecordPageMetaTx(ctx, tx, record)
	if err != nil {
		return PageMetaResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PageMetaResult{}, fmt.Errorf("%w: commit: %v", ErrDatabaseUnavailable, err)
	}
	return result, nil
}

// RecordPageMetaTx performs recognition and insertion inside the caller's transaction.
//
//  1. Serialize submissions for the same page (method + route URI) with a
//     transaction-scoped advisory lock; hash has no unique constraint.
//  2. Same structural hash already recorded → known (no write).
//  3. Page already recorded with a different hash → configuration_changed
//     (no write: the table has no candidate/approval state).
//  4. Otherwise insert the package; uuid and timestamps come from column defaults.
func RecordPageMetaTx(ctx context.Context, tx pgx.Tx, record PageMetaRecord) (PageMetaResult, error) {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, PageLockKey(record.Method, record.URI)); err != nil {
		return PageMetaResult{}, fmt.Errorf("%w: lock: %v", ErrDatabaseUnavailable, err)
	}

	known, err := queryPackages(ctx, tx, `
		SELECT uuid::text, hash, tiu::text, created_at, updated_at
		FROM pagemeta_packages
		WHERE hash = $1
		ORDER BY created_at, uuid
		LIMIT 1`, record.Hash)
	if err != nil {
		return PageMetaResult{}, err
	}
	if len(known) > 0 {
		return PageMetaResult{Result: PageMetaKnown, Package: &known[0], KnownPackages: []PageMetaPackage{}}, nil
	}

	samePage, err := queryPackages(ctx, tx, `
		SELECT uuid::text, hash, tiu::text, created_at, updated_at
		FROM pagemeta_packages
		WHERE payload->'data'->>'method' = $1
		  AND payload->'data'->'route'->>'uri' = $2
		ORDER BY created_at, uuid`, record.Method, record.URI)
	if err != nil {
		return PageMetaResult{}, err
	}
	if len(samePage) > 0 {
		return PageMetaResult{Result: PageMetaConfigurationChanged, KnownPackages: samePage}, nil
	}

	var inserted PageMetaPackage
	err = tx.QueryRow(ctx, `
		INSERT INTO pagemeta_packages (payload, hash, tiu)
		VALUES ($1::jsonb, $2, $3::numeric)
		RETURNING uuid::text, hash, tiu::text, created_at, updated_at`,
		string(record.Payload), record.Hash, record.TIU,
	).Scan(&inserted.UUID, &inserted.Hash, &inserted.TIU, &inserted.CreatedAt, &inserted.UpdatedAt)
	if err != nil {
		return PageMetaResult{}, fmt.Errorf("%w: insert: %v", ErrDatabaseUnavailable, err)
	}
	return PageMetaResult{Result: PageMetaRecorded, Recorded: true, Package: &inserted, KnownPackages: []PageMetaPackage{}}, nil
}

// PageLockKey derives the advisory lock key for a page identity.
func PageLockKey(method, uri string) int64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte("codexzone.pagemeta\x00" + method + "\x00" + uri))
	return int64(hasher.Sum64())
}

func queryPackages(ctx context.Context, tx pgx.Tx, sql string, args ...any) ([]PageMetaPackage, error) {
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: query: %v", ErrDatabaseUnavailable, err)
	}
	defer rows.Close()

	packages := []PageMetaPackage{}
	for rows.Next() {
		var p PageMetaPackage
		if err := rows.Scan(&p.UUID, &p.Hash, &p.TIU, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("%w: scan: %v", ErrDatabaseUnavailable, err)
		}
		packages = append(packages, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: rows: %v", ErrDatabaseUnavailable, err)
	}
	return packages, nil
}
