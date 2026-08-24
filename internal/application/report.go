package application

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"time"
)

func (s *EvidenceService) Report(ctx context.Context, scope domain.Scope) (domain.ComplianceReport, error) {
	evidence, err := s.evidence.ListEvidence(ctx, scope)
	if err != nil {
		return domain.ComplianceReport{}, err
	}
	batches, err := s.batches.ListBatches(ctx, scope)
	if err != nil {
		return domain.ComplianceReport{}, err
	}
	return domain.NewComplianceReport(scope, evidence, batches, time.Now())
}

type ReviewQueueItem struct {
	Batch         domain.ReviewBatch    `json:"batch"`
	Risk          domain.RiskAssessment `json:"risk"`
	EvidenceCount int                   `json:"evidence_count"`
}

func (s *EvidenceService) ReviewQueue(ctx context.Context, scope domain.Scope) ([]ReviewQueueItem, error) {
	batches, err := s.batches.ListBatches(ctx, scope)
	if err != nil {
		return nil, err
	}
	items := make([]ReviewQueueItem, 0)
	for _, batch := range batches {
		if batch.State != domain.BatchSubmitted && batch.State != domain.BatchUnderReview {
			continue
		}
		evidence, err := s.resolveEvidence(ctx, batch.Scope, batch.EvidenceIDs)
		if err != nil {
			return nil, err
		}
		items = append(items, ReviewQueueItem{Batch: batch, Risk: domain.AssessRisk(evidence, batch), EvidenceCount: len(evidence)})
	}
	return items, nil
}
