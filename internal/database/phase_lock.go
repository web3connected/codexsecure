package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/web3connected/codexsecure-api/internal/protect"
)

// Phase Lock pattern storage (CodexSecure P3; PL-2026-11-15, PLA-2026-11-15; schema P2).
//
// Multiple independent Phase Lock patterns per POST communication relationship. A
// relationship is zone + destination route (name, or URI when unnamed) + method; a
// structure package is one values-free canonical form structure of that relationship with
// its CodexHash v2 evidence and TIU; a pattern is one package recognized under an
// approval authority:
//
//	developer_admin       Developer Core: created pending; approved or rejected only by the
//	                      administrator decision of a later phase (never here)
//	communication_object  Communications decides at each request; no state is stored
//
// Only public.form_structure_packages and public.phase_lock_patterns are touched, as
// codex_zones_runtime: SELECT, and INSERT of the columns P2 granted. Nothing is updated
// or deleted, no state or decision column is written, and phase_lock_pattern_events is
// never written. Every rule (values-free documents, relationship agreement, initial
// states, immutability) is also enforced by the schema itself.

// Store outcomes, as the Laravel Core FormStructureStore contract names them.
const (
	PhaseLockRecorded   = "recorded"
	PhaseLockKnown      = "known"
	PhaseLockFound      = "found"
	PhaseLockNotFound   = "not_found"
	PhaseLockDiscovered = "discovered"
	PhaseLockExisting   = "existing"
)

// Approval authorities (phase_lock_patterns.authority).
const (
	PhaseLockDeveloperAdmin      = "developer_admin"
	PhaseLockCommunicationObject = "communication_object"
)

var (
	// ErrPhaseLockStoreUnavailable: no database connection or a database failure.
	ErrPhaseLockStoreUnavailable = errors.New("phase lock store unavailable")
	// ErrPhaseLockStructureRejected: the schema refused the package (it is not a valid
	// values-free structure of the relationship). The handler validates first, so this is
	// a second line of defence.
	ErrPhaseLockStructureRejected = errors.New("form structure rejected by the phase lock schema")
	// ErrPhaseLockStructureConflict: the package stored under this relationship and
	// CodexHash does not reproduce the verified canonical structure.
	ErrPhaseLockStructureConflict = errors.New("stored form structure does not reproduce the verified structure")
	// ErrPhaseLockAuthorityInvalid: not one of the two authorities.
	ErrPhaseLockAuthorityInvalid = errors.New("phase lock authority is invalid")
)

// PhaseLockRelationship is one POST communication relationship.
type PhaseLockRelationship struct {
	Zone   string `json:"zone"`
	Route  string `json:"route"`
	Method string `json:"method"`
}

// FormStructureRecord is a captured structure whose CodexHash v2 evidence the caller has
// already recomputed through the proprietary engine over Canonical.
type FormStructureRecord struct {
	Relationship PhaseLockRelationship
	// Canonical is the verified codexsecure-protect-payload-v1 form of the structure
	// document: the exact text the CodexHash v2 evidence was computed over.
	Canonical string
	CodexHash string
	// TIU is the CodexTime TIU as a decimal string (numeric(20,6)).
	TIU string
}

// FormStructurePackage is a stored structure package.
type FormStructurePackage struct {
	ID           int64
	Relationship PhaseLockRelationship
	// Data is the stored structure document in canonical form.
	Data      json.RawMessage
	CodexHash string
	TIU       string
	CreatedAt time.Time
}

// PhaseLockPattern is a stored pattern. State is nil for a Communications-governed
// pattern: its authorization is Communications' current answer, never a stored copy.
type PhaseLockPattern struct {
	ID                 int64
	Relationship       PhaseLockRelationship
	StructureCodexHash string
	Authority          string
	State              *string
	CreatedAt          time.Time
}

// PhaseLockEntry is one package of a relationship with its pattern, when discovered.
type PhaseLockEntry struct {
	Package FormStructurePackage
	Pattern *PhaseLockPattern
}

// RecordFormStructure records a verified structure in its own transaction.
func (db *DB) RecordFormStructure(ctx context.Context, record FormStructureRecord) (string, FormStructurePackage, error) {
	if db == nil || db.Pool == nil {
		return "", FormStructurePackage{}, ErrPhaseLockStoreUnavailable
	}
	tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", FormStructurePackage{}, fmt.Errorf("%w: begin: %v", ErrPhaseLockStoreUnavailable, err)
	}
	defer tx.Rollback(ctx)

	result, pkg, err := RecordFormStructureTx(ctx, tx, record)
	if err != nil {
		return "", FormStructurePackage{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", FormStructurePackage{}, fmt.Errorf("%w: commit: %v", ErrPhaseLockStoreUnavailable, err)
	}
	return result, pkg, nil
}

// RecordFormStructureTx keeps one immutable package per relationship and CodexHash.
//
//  1. Insert the package; the unique (zone, route_name, http_method,
//     structure_codexhash) constraint makes a repeat, or a concurrent duplicate, a
//     no-op (ON CONFLICT DO NOTHING waits for a concurrent insert of the same
//     identity, so exactly one row is ever created) → recorded.
//  2. Otherwise read the existing package → known. Its creation TIU is kept.
//  3. Either way the stored jsonb must reproduce the verified canonical text exactly,
//     so the stored document is the CodexHash v2 input (no numbers can be stored, and
//     canonicalization re-sorts keys and re-encodes strings). A difference refuses the
//     operation and rolls back.
//
// A different structure has a different CodexHash and becomes its own package: no
// package is ever updated, replaced or retired.
func RecordFormStructureTx(ctx context.Context, tx pgx.Tx, record FormStructureRecord) (string, FormStructurePackage, error) {
	rel := record.Relationship
	pkg := FormStructurePackage{Relationship: rel, CodexHash: record.CodexHash}
	var stored string
	result := PhaseLockRecorded
	err := tx.QueryRow(ctx, `
		INSERT INTO form_structure_packages (zone, route_name, http_method, structure_codexhash, data, tiu)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6::numeric)
		ON CONFLICT ON CONSTRAINT form_structure_packages_structure_unique DO NOTHING
		RETURNING id, data::text, tiu::text, created_at`,
		rel.Zone, rel.Route, rel.Method, record.CodexHash, record.Canonical, record.TIU,
	).Scan(&pkg.ID, &stored, &pkg.TIU, &pkg.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		result = PhaseLockKnown
		err = tx.QueryRow(ctx, `
			SELECT id, data::text, tiu::text, created_at
			FROM form_structure_packages
			WHERE zone = $1 AND route_name = $2 AND http_method = $3 AND structure_codexhash = $4`,
			rel.Zone, rel.Route, rel.Method, record.CodexHash,
		).Scan(&pkg.ID, &stored, &pkg.TIU, &pkg.CreatedAt)
	}
	if err != nil {
		return "", FormStructurePackage{}, phaseLockFailure("record form structure", err)
	}

	canonical, err := protect.CanonicalPayload(json.RawMessage(stored))
	if err != nil || canonical != record.Canonical {
		return "", FormStructurePackage{}, ErrPhaseLockStructureConflict
	}
	pkg.Data = json.RawMessage(canonical)
	return result, pkg, nil
}

// RelationshipPatterns returns every package of one relationship, oldest first, each with
// its pattern when one has been discovered.
func (db *DB) RelationshipPatterns(ctx context.Context, rel PhaseLockRelationship) ([]PhaseLockEntry, error) {
	if db == nil || db.Pool == nil {
		return nil, ErrPhaseLockStoreUnavailable
	}
	rows, err := db.Pool.Query(ctx, `
		SELECT s.id, s.data::text, s.structure_codexhash, s.tiu::text, s.created_at,
		       p.id, p.authority, p.state, p.created_at
		FROM form_structure_packages s
		LEFT JOIN phase_lock_patterns p ON p.package_id = s.id
		WHERE s.zone = $1 AND s.route_name = $2 AND s.http_method = $3
		ORDER BY s.created_at, s.id`, rel.Zone, rel.Route, rel.Method)
	if err != nil {
		return nil, phaseLockFailure("read relationship patterns", err)
	}
	defer rows.Close()

	entries := []PhaseLockEntry{}
	for rows.Next() {
		var (
			stored      string
			pkg         = FormStructurePackage{Relationship: rel}
			patternID   *int64
			authority   *string
			state       *string
			patternTime *time.Time
		)
		if err := rows.Scan(&pkg.ID, &stored, &pkg.CodexHash, &pkg.TIU, &pkg.CreatedAt,
			&patternID, &authority, &state, &patternTime); err != nil {
			return nil, phaseLockFailure("scan relationship patterns", err)
		}
		canonical, err := protect.CanonicalPayload(json.RawMessage(stored))
		if err != nil {
			return nil, ErrPhaseLockStructureConflict
		}
		pkg.Data = json.RawMessage(canonical)
		entry := PhaseLockEntry{Package: pkg}
		if patternID != nil {
			entry.Pattern = &PhaseLockPattern{ID: *patternID, Relationship: rel, StructureCodexHash: pkg.CodexHash,
				Authority: *authority, State: state, CreatedAt: *patternTime}
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, phaseLockFailure("read relationship patterns", err)
	}
	return entries, nil
}

// Pattern returns one pattern by its identity, or nil when none has been discovered.
func (db *DB) Pattern(ctx context.Context, rel PhaseLockRelationship, structureCodexHash string) (*PhaseLockPattern, error) {
	if db == nil || db.Pool == nil {
		return nil, ErrPhaseLockStoreUnavailable
	}
	return queryPattern(ctx, db.Pool, rel, structureCodexHash)
}

// DiscoverPattern discovers a pattern in its own transaction.
func (db *DB) DiscoverPattern(ctx context.Context, rel PhaseLockRelationship, structureCodexHash, authority string) (string, *PhaseLockPattern, error) {
	if db == nil || db.Pool == nil {
		return "", nil, ErrPhaseLockStoreUnavailable
	}
	tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", nil, fmt.Errorf("%w: begin: %v", ErrPhaseLockStoreUnavailable, err)
	}
	defer tx.Rollback(ctx)

	result, pattern, err := DiscoverPatternTx(ctx, tx, rel, structureCodexHash, authority)
	if err != nil {
		return "", nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", nil, fmt.Errorf("%w: commit: %v", ErrPhaseLockStoreUnavailable, err)
	}
	return result, pattern, nil
}

// DiscoverPatternTx turns a package already recorded for the relationship into a pattern.
//
//  1. No package for the relationship and CodexHash → not_found: a submission alone never
//     creates a structure or a pattern.
//  2. Insert the pattern with only its identity and authority; the schema sets the
//     initial state (pending for developer_admin, none for communication_object) and
//     refuses any decision. A repeat or concurrent duplicate is a no-op → discovered once.
//  3. Otherwise the existing pattern is returned unchanged → existing, whatever its
//     authority or state (PL-17: a change of governance never rewrites a stored pattern).
func DiscoverPatternTx(ctx context.Context, tx pgx.Tx, rel PhaseLockRelationship, structureCodexHash, authority string) (string, *PhaseLockPattern, error) {
	if authority != PhaseLockDeveloperAdmin && authority != PhaseLockCommunicationObject {
		return "", nil, ErrPhaseLockAuthorityInvalid
	}
	var packageID int64
	err := tx.QueryRow(ctx, `
		SELECT id FROM form_structure_packages
		WHERE zone = $1 AND route_name = $2 AND http_method = $3 AND structure_codexhash = $4`,
		rel.Zone, rel.Route, rel.Method, structureCodexHash).Scan(&packageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return PhaseLockNotFound, nil, nil
	}
	if err != nil {
		return "", nil, phaseLockFailure("read form structure", err)
	}

	pattern := PhaseLockPattern{Relationship: rel, StructureCodexHash: structureCodexHash}
	err = tx.QueryRow(ctx, `
		INSERT INTO phase_lock_patterns (package_id, zone, route_name, http_method, structure_codexhash, authority)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT DO NOTHING
		RETURNING id, authority, state, created_at`,
		packageID, rel.Zone, rel.Route, rel.Method, structureCodexHash, authority,
	).Scan(&pattern.ID, &pattern.Authority, &pattern.State, &pattern.CreatedAt)
	if err == nil {
		return PhaseLockDiscovered, &pattern, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", nil, phaseLockFailure("discover pattern", err)
	}

	existing, err := queryPattern(ctx, tx, rel, structureCodexHash)
	if err != nil {
		return "", nil, err
	}
	if existing == nil {
		return "", nil, fmt.Errorf("%w: discovered pattern not readable", ErrPhaseLockStoreUnavailable)
	}
	return PhaseLockExisting, existing, nil
}

type phaseLockQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func queryPattern(ctx context.Context, q phaseLockQuerier, rel PhaseLockRelationship, structureCodexHash string) (*PhaseLockPattern, error) {
	pattern := PhaseLockPattern{Relationship: rel, StructureCodexHash: structureCodexHash}
	err := q.QueryRow(ctx, `
		SELECT id, authority, state, created_at
		FROM phase_lock_patterns
		WHERE zone = $1 AND route_name = $2 AND http_method = $3 AND structure_codexhash = $4`,
		rel.Zone, rel.Route, rel.Method, structureCodexHash,
	).Scan(&pattern.ID, &pattern.Authority, &pattern.State, &pattern.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, phaseLockFailure("read pattern", err)
	}
	return &pattern, nil
}

// phaseLockFailure separates a schema refusal of the data (check or not-null violation)
// from an unavailable store. The cause is wrapped for logs, never returned to clients.
func phaseLockFailure(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23514" || pgErr.Code == "23502" || pgErr.Code == "22P02" || pgErr.Code == "22001") {
		return fmt.Errorf("%w: %s: %s", ErrPhaseLockStructureRejected, operation, pgErr.ConstraintName)
	}
	return fmt.Errorf("%w: %s: %v", ErrPhaseLockStoreUnavailable, operation, err)
}
