package communications

import (
	"context"
	"encoding/json"
	"time"
)

type Registration struct {
	Key       string  `json:"registration_key"`
	Scope     string  `json:"scope"`
	Component *string `json:"component"`
	Page      *string `json:"page"`
}
type Structure struct {
	State    string   `json:"state"`
	Revision int64    `json:"revision"`
	Issues   []string `json:"issues"`
}
type Approved struct {
	Revision          int64     `json:"revision"`
	Document          Document  `json:"document"`
	StructureRevision int64     `json:"structure_revision"`
	ApprovedBy        string    `json:"approved_by"`
	ApprovedAt        time.Time `json:"approved_at"`
}
type Draft struct {
	Version         int64     `json:"version"`
	Document        Document  `json:"document"`
	BasedOnRevision *int64    `json:"based_on_revision"`
	UpdatedBy       string    `json:"updated_by"`
	UpdatedAt       time.Time `json:"updated_at"`
}
type Event struct {
	Decision          string    `json:"decision"`
	Actor             string    `json:"actor"`
	DecidedAt         time.Time `json:"decided_at"`
	Revision          *int64    `json:"revision"`
	PreviousStatus    string    `json:"previous_status"`
	ResultingStatus   string    `json:"resulting_status"`
	StructureRevision *int64    `json:"structure_revision"`
	Reason            *string   `json:"reason"`
}
type Revision struct {
	Revision          int64    `json:"revision"`
	Document          Document `json:"document"`
	Status            string   `json:"status"`
	StructureRevision *int64   `json:"structure_revision"`
}
type ObservedAssessment struct {
	Observation
	Evaluation
}
type Configuration struct {
	Registration   Registration         `json:"registration"`
	Structure      Structure            `json:"structure"`
	ApprovalStatus string               `json:"approval_status"`
	Effective      string               `json:"effective"`
	Approved       *Approved            `json:"approved"`
	Draft          *Draft               `json:"draft"`
	History        []Event              `json:"history"`
	Observations   []ObservedAssessment `json:"observations"`
}
type History struct {
	History   []Event    `json:"history"`
	Revisions []Revision `json:"revisions"`
}
type Coverage struct {
	Type string `json:"type"`
	Page string `json:"page,omitempty"`
}
type CoverageGap struct {
	Code            string `json:"code"`
	Action          string `json:"action"`
	RegistrationKey string `json:"registration_key,omitempty"`
	Page            string `json:"page,omitempty"`
}
type AssessmentProgress struct {
	RegisteredCommunications int `json:"registered_communications"`
	EvaluatedCommunications  int `json:"evaluated_communications"`
	EvaluatedObservations    int `json:"evaluated_observations"`
}
type FinalizedCoverage struct {
	Generation int64      `json:"generation"`
	BatchKey   string     `json:"batch_key"`
	CoverageAt *time.Time `json:"coverage_at"`
}
type Assessment struct {
	Page               *SourcePage        `json:"page,omitempty"`
	Coverage           *FinalizedCoverage `json:"coverage,omitempty"`
	Site               string             `json:"site"`
	Scope              Coverage           `json:"scope"`
	Complete           bool               `json:"complete"`
	AssessedAt         time.Time          `json:"assessed_at"`
	CoverageBasis      string             `json:"coverage_basis"`
	CompletenessIssues []string           `json:"completeness_issues"`
	Communications     []Configuration    `json:"communications"`
	CoverageGaps       []CoverageGap      `json:"coverage_gaps"`
	Progress           AssessmentProgress `json:"progress"`
}
type Decision struct {
	Action            string
	Site              string
	Key               string
	Actor             string
	IdempotencyKey    string
	Document          json.RawMessage
	ExpectedVersion   *int64
	DraftVersion      *int64
	StructureRevision *int64
	Reason            *string
}

// Store uses the existing PostgreSQL pool and authoritative decision functions.
// It is also the handler seam for unit tests; it is not another database layer.
type Store interface {
	Find(context.Context, string, string) (Configuration, error)
	History(context.Context, string, string) (History, error)
	Decide(context.Context, Decision) (json.RawMessage, error)
	Discover(context.Context, string, string, Discovery) (DiscoveryResult, error)
	Assess(context.Context, string, Coverage) (Assessment, error)
}

func conflict(code, message string) error { return APIError{Status: 409, Code: code, Message: message} }
