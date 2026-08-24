package domain

import (
	"fmt"
	"sort"
	"time"
)

type ComplianceReport struct {
	Scope       Scope          `json:"scope"`
	GeneratedAt time.Time      `json:"generated_at"`
	Evidence    []Evidence     `json:"evidence"`
	Batches     []ReviewBatch  `json:"batches"`
	Risk        RiskAssessment `json:"risk"`
}

func NewComplianceReport(scope Scope, evidence []Evidence, batches []ReviewBatch, now time.Time) (ComplianceReport, error) {
	if err := scope.Validate(); err != nil {
		return ComplianceReport{}, err
	}
	for _, item := range evidence {
		if item.Scope() != scope {
			return ComplianceReport{}, fmt.Errorf("evidence %s is outside report scope", item.ID)
		}
	}
	for _, batch := range batches {
		if batch.Scope != scope {
			return ComplianceReport{}, fmt.Errorf("batch %s is outside report scope", batch.ID)
		}
	}
	sort.Slice(evidence, func(left, right int) bool { return evidence[left].Reference < evidence[right].Reference })
	sort.Slice(batches, func(left, right int) bool { return batches[left].CreatedAt.Before(batches[right].CreatedAt) })
	risk := RiskAssessment{Level: RiskLow, Reasons: []string{}}
	for _, batch := range batches {
		candidate := AssessRisk(evidence, batch)
		if candidate.Level == RiskHigh {
			risk = candidate
			break
		}
		if candidate.Level == RiskMedium {
			risk = candidate
		}
	}
	return ComplianceReport{Scope: scope, GeneratedAt: now.UTC(), Evidence: evidence, Batches: batches, Risk: risk}, nil
}

func (r ComplianceReport) Approved() int {
	count := 0
	for _, batch := range r.Batches {
		if batch.State == BatchApproved {
			count++
		}
	}
	return count
}
func (r ComplianceReport) Open() int {
	count := 0
	for _, batch := range r.Batches {
		if !batch.IsTerminal() {
			count++
		}
	}
	return count
}
func (r ComplianceReport) References() []string {
	values := make([]string, 0, len(r.Evidence))
	for _, item := range r.Evidence {
		values = append(values, item.Reference)
	}
	return values
}
