package communications

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresStore) Assess(ctx context.Context, site string, coverage Coverage) (result Assessment, err error) {
	result = Assessment{Site: site, Scope: coverage, AssessedAt: time.Now().UTC(), CoverageBasis: "finalized_generation", Communications: []Configuration{}, CompletenessIssues: []string{}, CoverageGaps: []CoverageGap{}}
	defer func() {
		if err != nil {
			result.Complete = false
			var api APIError
			var validation ValidationErrors
			if !errors.As(err, &validation) && (!errors.As(err, &api) || api.Status >= 500) {
				result.addGap(CoverageGap{Code: "assessment_processing_failed", Action: "Retry the declared scope after the database or processing failure is resolved; this response cannot resolve notifications", Page: coverage.Page})
			}
		}
	}()
	if coverage.Type != "site" && coverage.Type != "page" {
		return result, ValidationErrors{"scope": {"must be site or page"}}
	}
	if coverage.Type == "page" && !absoluteDestination(coverage.Page) || coverage.Type == "site" && coverage.Page != "" {
		return result, ValidationErrors{"page": {"page scope requires an absolute page URL; site scope has no page filter"}}
	}
	tx, e := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return result, e
	}
	defer tx.Rollback(ctx)
	var siteID int64
	var base string
	if e = tx.QueryRow(ctx, `SELECT id,base_url FROM public.sites WHERE site_key=$1`, site).Scan(&siteID, &base); e == pgx.ErrNoRows {
		return result, APIError{404, "site is not enrolled", "SITE_NOT_FOUND", nil}
	} else if e != nil {
		return result, e
	}
	if coverage.Type == "page" {
		if !sameOrigin(base, coverage.Page) {
			return result, ValidationErrors{"page": {"must belong to the registered site origin"}}
		}
		var page SourcePage
		if e = tx.QueryRow(ctx, `SELECT page_key,url,zone FROM public.pages WHERE site_id=$1 AND url=$2`, siteID, coverage.Page).Scan(&page.Key, &page.URL, &page.Zone); e == pgx.ErrNoRows {
			return result, APIError{404, "page has not been registered", "PAGE_NOT_FOUND", nil}
		} else if e != nil {
			return result, e
		}
		result.Page = &page
	}
	batchID, e := assessmentCoverage(ctx, tx, siteID, coverage, &result)
	if e != nil {
		return result, e
	}
	rows, e := tx.Query(ctx, `SELECT o.registration_key::text FROM public.communication_objects o WHERE o.site_id=$1 AND ($2::text='' OR EXISTS(SELECT 1 FROM public.pages p WHERE p.id=o.page_id AND p.url=$2) OR (o.scope='universal' AND EXISTS(SELECT 1 FROM public.communication_observations v JOIN public.pages p ON p.id=v.page_id AND p.site_id=v.site_id WHERE v.communication_object_id=o.id AND p.url=$2))) ORDER BY o.registration_key`, siteID, coverage.Page)
	if e != nil {
		return result, e
	}
	keys := []string{}
	for rows.Next() {
		var key string
		if e = rows.Scan(&key); e != nil {
			rows.Close()
			return result, e
		}
		keys = append(keys, key)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return result, e
	}
	result.Progress.RegisteredCommunications = len(keys)
	for _, key := range keys {
		c, e := s.findEvidenceTx(ctx, tx, site, key, coverage.Page, true)
		if e != nil {
			result.addGap(CoverageGap{Code: "communication_evaluation_failed", Action: "Retry evaluation of this registered communication; no result was inferred", RegistrationKey: key, Page: coverage.Page})
			return result, e
		}
		var accounted bool
		if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public.communication_discovery_absences a JOIN public.communication_objects o ON o.id=a.communication_object_id JOIN public.pages p ON p.id=a.page_id JOIN public.communication_page_coverage pc ON pc.covered_batch_id=a.batch_id AND pc.current WHERE o.registration_key=$1::uuid AND o.site_id=$2 AND ($3::text='' OR p.url=$3))`, key, siteID, coverage.Page).Scan(&accounted); e != nil {
			return result, e
		}
		if len(c.Observations) == 0 && !accounted {
			result.addGap(CoverageGap{Code: "communication_observations_missing", Action: "Ingest observations for this communication within the declared source-page scope", RegistrationKey: key, Page: coverage.Page})
		}
		if c.Structure.State == "unknown" {
			result.addGap(CoverageGap{Code: "structure_unassessed", Action: "Ingest and validate the current structural declarations", RegistrationKey: key, Page: coverage.Page})
		}
		result.Communications = append(result.Communications, c)
		result.Progress.EvaluatedCommunications++
		result.Progress.EvaluatedObservations += len(c.Observations)
	}
	// Absences are findings from a complete scan, not completeness failures.
	absenceRows, e := tx.Query(ctx, `SELECT a.absence,o.registration_key::text,p.url FROM public.communication_discovery_absences a JOIN public.communication_objects o ON o.id=a.communication_object_id JOIN public.pages p ON p.id=a.page_id JOIN public.communication_page_coverage pc ON pc.covered_batch_id=a.batch_id AND pc.current JOIN public.communication_discovery_batches b ON b.id=a.batch_id WHERE a.site_id=$1 AND (($2::text<>'' AND p.url=$2) OR ($2::text='' AND b.parent_batch_id=$3)) ORDER BY p.url,o.registration_key`, siteID, coverage.Page, batchID)
	if e != nil {
		return result, e
	}
	for absenceRows.Next() {
		var gap CoverageGap
		if e = absenceRows.Scan(&gap.Code, &gap.RegistrationKey, &gap.Page); e != nil {
			absenceRows.Close()
			return result, e
		}
		gap.Action = "Review communication absent from this complete discovery generation"
		result.CoverageGaps = append(result.CoverageGaps, gap)
	}
	e = absenceRows.Err()
	absenceRows.Close()
	if e != nil {
		return result, e
	}
	result.Complete = result.Coverage != nil && len(result.CompletenessIssues) == 0 && result.Progress.EvaluatedCommunications == result.Progress.RegisteredCommunications

	result.AssessedAt = time.Now().UTC()
	return result, tx.Commit(ctx)
}

func (a *Assessment) addGap(gap CoverageGap) {
	for _, code := range a.CompletenessIssues {
		if code == gap.Code {
			a.CoverageGaps = append(a.CoverageGaps, gap)
			return
		}
	}
	a.CompletenessIssues = append(a.CompletenessIssues, gap.Code)
	a.CoverageGaps = append(a.CoverageGaps, gap)
}

// All checks and evidence reads use the same repeatable-read snapshot.
func assessmentCoverage(ctx context.Context, tx pgx.Tx, siteID int64, scope Coverage, result *Assessment) (*int64, error) {
	var id int64
	var generation int64
	var key, status string
	var at, invalidated *time.Time
	var issues []byte
	e := tx.QueryRow(ctx, `SELECT b.id,b.generation,b.batch_key::text,b.status,b.coverage_at,b.invalidated_at,b.completeness_issues FROM public.communication_discovery_batches b LEFT JOIN public.pages p ON p.id=b.page_id WHERE b.site_id=$1 AND b.scope=$2 AND ($2='site' OR p.url=$3) ORDER BY b.generation DESC LIMIT 1`, siteID, scope.Type, scope.Page).Scan(&id, &generation, &key, &status, &at, &invalidated, &issues)
	if e == pgx.ErrNoRows {
		result.addGap(CoverageGap{Code: "discovery_generation_missing", Page: scope.Page, Action: "Finalize discovery for this scope"})
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	result.Coverage = &FinalizedCoverage{Generation: generation, BatchKey: key, CoverageAt: at}
	if status != "complete" {
		code := "discovery_generation_" + status
		if status == "failed" {
			code = "discovery_generation_incomplete"
		}
		result.addGap(CoverageGap{Code: code, Page: scope.Page, Action: "Complete a new discovery generation"})
	}
	if invalidated != nil {
		result.addGap(CoverageGap{Code: "coverage_invalidated", Page: scope.Page, Action: "Finalize a new discovery generation"})
	}
	var details []struct {
		Code string `json:"code"`
		Page string `json:"page"`
	}
	if e = json.Unmarshal(issues, &details); e != nil {
		return nil, e
	}
	for _, issue := range details {
		result.addGap(CoverageGap{Code: issue.Code, Page: issue.Page, Action: "Resolve discovery coverage issue"})
	}
	if scope.Type == "site" {
		rows, e := tx.Query(ctx, `SELECT m.page_key FROM public.communication_discovery_batch_pages m WHERE m.batch_id=$1 AND NOT EXISTS(SELECT 1 FROM public.communication_discovery_batches b JOIN public.pages p ON p.id=b.page_id JOIN public.communication_page_coverage c ON c.covered_batch_id=b.id AND c.current WHERE b.parent_batch_id=$1 AND p.page_key=m.page_key) UNION SELECT p.page_key FROM public.pages p WHERE p.site_id=$2 AND NOT EXISTS(SELECT 1 FROM public.communication_discovery_batch_pages m WHERE m.batch_id=$1 AND m.page_key=p.page_key)`, id, siteID)
		if e != nil {
			return nil, e
		}
		for rows.Next() {
			var page string
			if e = rows.Scan(&page); e != nil {
				rows.Close()
				return nil, e
			}
			result.addGap(CoverageGap{Code: "page_not_covered", Page: page, Action: "Include and complete this page in the site generation"})
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
	} else {
		var current bool
		if e = tx.QueryRow(ctx, `SELECT current FROM public.communication_page_coverage WHERE site_id=$1 AND page_url=$2`, siteID, scope.Page).Scan(&current); e != nil {
			return nil, e
		}
		if !current && status == "complete" && invalidated == nil {
			result.addGap(CoverageGap{Code: "page_not_covered", Page: scope.Page, Action: "Complete current page discovery"})
		}
	}
	return &id, nil
}
