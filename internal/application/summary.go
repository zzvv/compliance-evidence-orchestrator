package application

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

func (s *EvidenceService) ScopeSummary(ctx context.Context, scope domain.Scope) (domain.ScopeSummary, error) {
	evidence, err := s.evidence.ListEvidence(ctx, scope)
	if err != nil {
		return domain.ScopeSummary{}, err
	}
	batches, err := s.batches.ListBatches(ctx, scope)
	if err != nil {
		return domain.ScopeSummary{}, err
	}
	return domain.SummarizeScope(scope, evidence, batches), nil
}
