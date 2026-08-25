package domain

import (
	"fmt"
	"strings"
	"time"
)

type EvidenceKind string

const (
	Certificate EvidenceKind = "certificate"
	TestReport  EvidenceKind = "test_report"
	Declaration EvidenceKind = "declaration"
)

type Evidence struct {
	ID         string       `json:"id"`
	ProjectID  string       `json:"project_id"`
	MaterialID string       `json:"material_id"`
	Reference  string       `json:"reference"`
	Kind       EvidenceKind `json:"kind"`
	Supplier   string       `json:"supplier"`
	IssuedAt   time.Time    `json:"issued_at"`
	ExpiresAt  *time.Time   `json:"expires_at,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
}

func NewEvidence(id, projectID, materialID, reference string, kind EvidenceKind, supplier string, issuedAt time.Time) (Evidence, error) {
	evidence := Evidence{ID: strings.TrimSpace(id), ProjectID: strings.TrimSpace(projectID), MaterialID: strings.TrimSpace(materialID), Reference: strings.TrimSpace(reference), Kind: kind, Supplier: strings.TrimSpace(supplier), IssuedAt: issuedAt.UTC(), CreatedAt: time.Now().UTC()}
	if err := evidence.Validate(); err != nil {
		return Evidence{}, err
	}
	return evidence, nil
}

func (e Evidence) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("evidence id is required")
	}
	if e.ProjectID == "" {
		return fmt.Errorf("project id is required")
	}
	if e.MaterialID == "" {
		return fmt.Errorf("material id is required")
	}
	if e.Reference == "" {
		return fmt.Errorf("evidence reference is required")
	}
	if e.Supplier == "" {
		return fmt.Errorf("supplier is required")
	}
	if e.IssuedAt.IsZero() {
		return fmt.Errorf("issued time is required")
	}
	switch e.Kind {
	case Certificate, TestReport, Declaration:
	default:
		return fmt.Errorf("unsupported evidence kind %q", e.Kind)
	}
	if e.ExpiresAt != nil && e.ExpiresAt.Before(e.IssuedAt) {
		return fmt.Errorf("expiry precedes issue date")
	}
	return nil
}

func (e Evidence) IsExpired(at time.Time) bool {
	return e.ExpiresAt != nil && !e.ExpiresAt.After(at.UTC())
}
func (e Evidence) Scope() Scope { return Scope{ProjectID: e.ProjectID, MaterialID: e.MaterialID} }
