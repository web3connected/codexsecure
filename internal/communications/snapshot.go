package communications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
	"github.com/web3connected/codexsecure-api/internal/protect"
	"github.com/web3connected/codexsecure-api/pkg/codexhash"
)

// Canonical forms hashed by CodexHash, both under the codexsecure-protect-payload-v1
// rules (protect.CanonicalPayload). docs/codexsecure/COMMUNICATION_SNAPSHOT_C13E.md is
// the contract Web3Codex reproduces.
//
// codexsecure-approved-communication-v1:
//
//	{"approved_revision","communication_uuid","component_key","destination",
//	 "destination_match","enabled","methods","page_key","representation","scope",
//	 "site_key","type","zone"}
//
// codexsecure-approved-snapshot-v1:
//
//	{"communications":[{"codex_hash","communication_uuid"},...],"representation",
//	 "site_key","snapshot_version"}
//
// Communications are ordered by communication_uuid. No timestamp is part of either.
const (
	ConfigurationRepresentation = "codexsecure-approved-communication-v1"
	SnapshotRepresentation      = "codexsecure-approved-snapshot-v1"
)

const (
	snapshotHashAlgo = "sha512"
	snapshotHashSize = 512
	// DefaultSnapshotTTL bounds how long Web3Codex may treat a read as current.
	DefaultSnapshotTTL = 5 * time.Minute
	// A retry is only needed when another reader or a decision publishes first.
	snapshotPublishAttempts = 3
)

var snapshotHashFormat = regexp.MustCompile(`^[0-9a-f]{128}$`)

// SnapshotCommunication is one approved configuration. Its fields are the local
// codex_communication_approvals columns Web3Codex fills from it.
type SnapshotCommunication struct {
	CommunicationUUID string   `json:"communication_uuid"`
	Scope             string   `json:"scope"`
	PageKey           *string  `json:"page_key"`
	ComponentKey      *string  `json:"component_key"`
	Type              string   `json:"type"`
	Destination       string   `json:"destination"`
	DestinationMatch  string   `json:"destination_match"`
	Methods           []string `json:"methods"`
	Zone              string   `json:"zone"`
	Enabled           bool     `json:"enabled"`
	ApprovedRevision  int64    `json:"approved_revision"`
	CodexHash         string   `json:"codex_hash"`
}
type SnapshotPage struct {
	PageKey        string                  `json:"page_key"`
	Communications []SnapshotCommunication `json:"communications"`
}
type SnapshotCommunications struct {
	Universal []SnapshotCommunication `json:"universal"`
	Pages     []SnapshotPage          `json:"pages"`
	Admin     []SnapshotCommunication `json:"admin"`
}
type Snapshot struct {
	SiteKey         string                 `json:"site_key"`
	SnapshotVersion int64                  `json:"snapshot_version"`
	GeneratedAt     time.Time              `json:"snapshot_generated_at"`
	ExpiresAt       time.Time              `json:"expires_at"`
	SnapshotHash    string                 `json:"snapshot_hash"`
	Communications  SnapshotCommunications `json:"communications"`
}

// SnapshotStore keeps the capability separate from configuration stores, like BatchStore.
type SnapshotStore interface {
	Snapshot(context.Context, string) (Snapshot, error)
}

// PayloadHasher is the CodexHash operation used for snapshots (codexhash.Client).
// Hash and QuickHash are deliberately absent: they can fall back to a local hash.
type PayloadHasher interface {
	PayloadHash(ctx context.Context, data string, algo string, hashSize int) (codexhash.PayloadHashResult, error)
}

// WithSnapshots enables Snapshot on a store.
func (s *PostgresStore) WithSnapshots(hasher PayloadHasher, ttl time.Duration) *PostgresStore {
	s.hasher = hasher
	s.snapshotTTL = ttl
	return s
}

// SnapshotTTLFromEnv reads COMMUNICATION_SNAPSHOT_TTL_SECONDS (whole positive seconds).
func SnapshotTTLFromEnv() time.Duration {
	if seconds, e := strconv.Atoi(os.Getenv("COMMUNICATION_SNAPSHOT_TTL_SECONDS")); e == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return DefaultSnapshotTTL
}

type snapshotSourceRow struct {
	CommunicationUUID         string          `json:"communication_uuid"`
	Scope                     string          `json:"scope"`
	PageKey                   *string         `json:"page_key"`
	ComponentKey              *string         `json:"component_key"`
	ApprovalStatus            string          `json:"approval_status"`
	StructureState            string          `json:"structure_state"`
	StructureRevision         int64           `json:"structure_revision"`
	ApprovedStructureRevision int64           `json:"approved_structure_revision"`
	ApprovedRevision          int64           `json:"approved_revision"`
	Document                  json.RawMessage `json:"document"`
}
type snapshotExclusion struct {
	uuid      string
	effective string
}

// snapshotItem is one included configuration with the exact text its hash covers.
type snapshotItem struct {
	communication SnapshotCommunication
	canonical     string
}
type storedSnapshotItem struct {
	canonical string
	hash      string
}

// snapshotRead is one consistent view of the source and the current published state.
type snapshotRead struct {
	site        string
	source      []byte
	items       []snapshotItem
	excluded    []snapshotExclusion
	version     int64
	hash        string
	generatedAt time.Time
	stored      map[string]storedSnapshotItem
}

// current reports whether the published version still describes the approved state.
func (r snapshotRead) current() bool {
	if r.version == 0 || len(r.items) != len(r.stored) {
		return false
	}
	for _, item := range r.items {
		stored, found := r.stored[item.communication.CommunicationUUID]
		if !found || stored.canonical != item.canonical {
			return false
		}
	}
	return true
}

// Snapshot returns the site's approved configuration. Discovery observations are never
// read. A new version is published only when the approved state differs from the one
// published, so repeated reads return the same version and hash without calling CodexHash.
// If a change cannot be hashed the read fails: an outdated version is never served as current.
func (s *PostgresStore) Snapshot(ctx context.Context, site string) (Snapshot, error) {
	if s.hasher == nil {
		return Snapshot{}, APIError{Status: 503, Code: "SNAPSHOT_UNAVAILABLE", Message: "snapshot service unavailable"}
	}
	for attempt := 0; attempt < snapshotPublishAttempts; attempt++ {
		read, e := s.readSnapshot(ctx, site)
		if e != nil {
			return Snapshot{}, e
		}
		if read.current() {
			for i := range read.items {
				read.items[i].communication.CodexHash = read.stored[read.items[i].communication.CommunicationUUID].hash
			}
			return s.assembleSnapshot(read)
		}
		if e = s.publishSnapshot(ctx, &read); e != nil {
			var database *pgconn.PgError
			if errors.As(e, &database) && database.Code == "CX409" {
				continue
			}
			return Snapshot{}, e
		}
		return s.assembleSnapshot(read)
	}
	return Snapshot{}, APIError{Status: 503, Code: "SNAPSHOT_UNAVAILABLE", Message: "the approved configuration is changing; retry"}
}

func (s *PostgresStore) readSnapshot(ctx context.Context, site string) (snapshotRead, error) {
	read := snapshotRead{site: site, stored: map[string]storedSnapshotItem{}}
	tx, e := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return read, e
	}
	defer tx.Rollback(ctx)
	if e = tx.QueryRow(ctx, `SELECT public.codex_communications_snapshot_source($1::text)`, site).Scan(&read.source); e != nil {
		return read, e
	}
	if read.source == nil {
		return read, APIError{404, "site is not enrolled", "SITE_NOT_FOUND", nil}
	}
	e = tx.QueryRow(ctx, `SELECT c.snapshot_version,c.snapshot_hash,c.generated_at FROM public.communication_snapshots c JOIN public.sites s ON s.id=c.site_id WHERE s.site_key=$1`, site).Scan(&read.version, &read.hash, &read.generatedAt)
	if e != nil && e != pgx.ErrNoRows {
		return read, e
	}
	rows, e := tx.Query(ctx, `SELECT o.registration_key::text,i.canonical_configuration,i.codex_hash FROM public.communication_snapshot_items i JOIN public.sites s ON s.id=i.site_id JOIN public.communication_objects o ON o.id=i.communication_object_id AND o.site_id=i.site_id WHERE s.site_key=$1`, site)
	if e != nil {
		return read, e
	}
	for rows.Next() {
		var key string
		var stored storedSnapshotItem
		if e = rows.Scan(&key, &stored.canonical, &stored.hash); e != nil {
			rows.Close()
			return read, e
		}
		read.stored[key] = stored
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return read, e
	}
	if e = tx.Commit(ctx); e != nil {
		return read, e
	}
	return read, s.evaluateSnapshotSource(&read)
}

// evaluateSnapshotSource applies the authority's own evaluation to each stored approval,
// using the approved configuration in place of an observation. An enabled configuration is
// included only when that evaluates to approved, which includes the destination-zone rule.
// A disabled one is included with enabled=false, so its withdrawal is explicit. Suspended,
// reapproval-required and otherwise unauthorized configurations are left out.
func (s *PostgresStore) evaluateSnapshotSource(read *snapshotRead) error {
	var source struct {
		SiteKey        string              `json:"site_key"`
		BaseURL        string              `json:"base_url"`
		Communications []snapshotSourceRow `json:"communications"`
	}
	if e := json.Unmarshal(read.source, &source); e != nil {
		return e
	}
	sort.Slice(source.Communications, func(a, b int) bool {
		return source.Communications[a].CommunicationUUID < source.Communications[b].CommunicationUUID
	})
	for _, row := range source.Communications {
		document, e := Validate(row.Document, s.zones)
		if e != nil {
			return fmt.Errorf("stored approved configuration invalid")
		}
		probe := Observation{Destination: document.Destination, Method: document.Methods[0], Type: document.Type, Zone: document.Zone}
		evaluation := Evaluate(row.ApprovalStatus, row.StructureState, row.StructureRevision, row.ApprovedStructureRevision, &document, probe, source.BaseURL)
		if evaluation.Effective != "approved" && evaluation.Effective != "disabled" {
			read.excluded = append(read.excluded, snapshotExclusion{row.CommunicationUUID, evaluation.Effective})
			continue
		}
		communication := SnapshotCommunication{
			CommunicationUUID: row.CommunicationUUID,
			Scope:             row.Scope,
			PageKey:           row.PageKey,
			ComponentKey:      row.ComponentKey,
			Type:              document.Type,
			Destination:       document.Destination,
			DestinationMatch:  document.DestinationMatch,
			Methods:           document.Methods,
			Zone:              document.Zone,
			Enabled:           evaluation.Effective == "approved",
			ApprovedRevision:  row.ApprovedRevision,
		}
		canonical, e := canonicalConfiguration(source.SiteKey, communication)
		if e != nil {
			return e
		}
		read.items = append(read.items, snapshotItem{communication, canonical})
	}
	read.site = source.SiteKey
	return nil
}

func canonicalConfiguration(site string, c SnapshotCommunication) (string, error) {
	raw, e := json.Marshal(map[string]any{
		"approved_revision":  c.ApprovedRevision,
		"communication_uuid": c.CommunicationUUID,
		"component_key":      c.ComponentKey,
		"destination":        c.Destination,
		"destination_match":  c.DestinationMatch,
		"enabled":            c.Enabled,
		"methods":            c.Methods,
		"page_key":           c.PageKey,
		"representation":     ConfigurationRepresentation,
		"scope":              c.Scope,
		"site_key":           site,
		"type":               c.Type,
		"zone":               c.Zone,
	})
	if e != nil {
		return "", e
	}
	return protect.CanonicalPayload(raw)
}

func canonicalSnapshot(site string, version int64, items []snapshotItem) (string, error) {
	members := make([]map[string]string, 0, len(items))
	for _, item := range items {
		members = append(members, map[string]string{"codex_hash": item.communication.CodexHash, "communication_uuid": item.communication.CommunicationUUID})
	}
	raw, e := json.Marshal(map[string]any{
		"communications":   members,
		"representation":   SnapshotRepresentation,
		"site_key":         site,
		"snapshot_version": version,
	})
	if e != nil {
		return "", e
	}
	return protect.CanonicalPayload(raw)
}

func (s *PostgresStore) snapshotHash(ctx context.Context, canonical string) (string, error) {
	result, e := s.hasher.PayloadHash(ctx, canonical, snapshotHashAlgo, snapshotHashSize)
	if e != nil || result.Algo != snapshotHashAlgo || !snapshotHashFormat.MatchString(result.PayloadHash) {
		log.Warn().Str("operation", "communications.snapshot").Msg("CodexHash unavailable for approved snapshot")
		return "", APIError{Status: 503, Code: "CODEXHASH_UNAVAILABLE", Message: "CodexHash could not hash the approved snapshot"}
	}
	return result.PayloadHash, nil
}

// publishSnapshot hashes outside any transaction, as channel activation does, reusing the
// stored hash of each unchanged configuration. The database function then refuses with
// CX409 if the source or version moved in the meantime.
func (s *PostgresStore) publishSnapshot(ctx context.Context, read *snapshotRead) error {
	items := make([]map[string]any, 0, len(read.items))
	for i := range read.items {
		item := &read.items[i]
		stored, found := read.stored[item.communication.CommunicationUUID]
		if found && stored.canonical == item.canonical {
			item.communication.CodexHash = stored.hash
		} else {
			hash, e := s.snapshotHash(ctx, item.canonical)
			if e != nil {
				return e
			}
			item.communication.CodexHash = hash
		}
		items = append(items, map[string]any{
			"communication_uuid":      item.communication.CommunicationUUID,
			"approved_revision":       item.communication.ApprovedRevision,
			"canonical_configuration": item.canonical,
			"codex_hash":              item.communication.CodexHash,
		})
	}
	next := read.version + 1
	canonical, e := canonicalSnapshot(read.site, next, read.items)
	if e != nil {
		return e
	}
	hash, e := s.snapshotHash(ctx, canonical)
	if e != nil {
		return e
	}
	encoded, e := json.Marshal(items)
	if e != nil {
		return e
	}
	var raw []byte
	e = s.pool.QueryRow(ctx, `SELECT public.codex_communications_snapshot_publish($1::text,$2::bigint,$3::jsonb,$4::text,$5::jsonb)`, read.site, read.version, string(read.source), hash, string(encoded)).Scan(&raw)
	if e != nil {
		return e
	}
	var published struct {
		SnapshotVersion int64     `json:"snapshot_version"`
		GeneratedAt     time.Time `json:"generated_at"`
	}
	if e = json.Unmarshal(raw, &published); e != nil {
		return e
	}
	read.version = published.SnapshotVersion
	read.generatedAt = published.GeneratedAt
	read.hash = hash
	for _, excluded := range read.excluded {
		log.Warn().Str("operation", "communications.snapshot").Str("site", read.site).Str("registration_key", excluded.uuid).Str("effective", excluded.effective).Msg("Stored approval left out of the approved snapshot")
	}
	return nil
}

// assembleSnapshot arranges an included, hashed read by scope. Items arrive ordered by
// communication_uuid; pages are ordered by page_key. expires_at is measured from this read:
// the version may be older, but it was confirmed current now.
func (s *PostgresStore) assembleSnapshot(read snapshotRead) (Snapshot, error) {
	result := Snapshot{
		SiteKey:         read.site,
		SnapshotVersion: read.version,
		GeneratedAt:     read.generatedAt.UTC().Truncate(time.Second),
		ExpiresAt:       time.Now().UTC().Truncate(time.Second).Add(s.snapshotTTL),
		SnapshotHash:    read.hash,
		Communications: SnapshotCommunications{
			Universal: []SnapshotCommunication{},
			Pages:     []SnapshotPage{},
			Admin:     []SnapshotCommunication{},
		},
	}
	pages := map[string][]SnapshotCommunication{}
	for _, item := range read.items {
		c := item.communication
		switch {
		case c.Scope == "universal":
			result.Communications.Universal = append(result.Communications.Universal, c)
		case c.Scope == "admin":
			result.Communications.Admin = append(result.Communications.Admin, c)
		case c.Scope == "page" && c.PageKey != nil:
			pages[*c.PageKey] = append(pages[*c.PageKey], c)
		default:
			return Snapshot{}, fmt.Errorf("stored communication scope invalid")
		}
	}
	for key, communications := range pages {
		result.Communications.Pages = append(result.Communications.Pages, SnapshotPage{key, communications})
	}
	sort.Slice(result.Communications.Pages, func(a, b int) bool {
		return result.Communications.Pages[a].PageKey < result.Communications.Pages[b].PageKey
	})
	return result, nil
}

var _ SnapshotStore = (*PostgresStore)(nil)
