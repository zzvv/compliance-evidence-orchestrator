package application

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

func (s *EvidenceService) Timeline(ctx context.Context, batchID string) ([]domain.TimelineEntry, error) {
	detail, err := s.BatchDetail(ctx, batchID)
	if err != nil {
		return nil, err
	}
	return domain.BuildTimeline(detail.Batch, detail.Receipts, detail.Audit), nil
}
func (s *EvidenceService) CanStartReview(ctx context.Context, batchID string) (bool, []string, error) {
	if err := s.ValidateBatch(ctx, batchID); err != nil {
		return false, []string{err.Error()}, nil
	}
	return true, nil, nil
}
