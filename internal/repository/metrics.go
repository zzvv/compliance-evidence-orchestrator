package repository

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

type Metrics struct {
	Evidence      int
	Batches       int
	Receipts      int
	Notifications int
	AuditEvents   int
}

func (s *Store) Metrics(ctx context.Context) (Metrics, error) {
	if err := ctx.Err(); err != nil {
		return Metrics{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	metrics := Metrics{Evidence: len(s.evidence), Batches: len(s.batches), Notifications: len(s.notifications)}
	for _, items := range s.receipts {
		metrics.Receipts += len(items)
	}
	for _, items := range s.audits {
		metrics.AuditEvents += len(items)
	}
	return metrics, nil
}
func (s *Store) Summary(ctx context.Context, scope domain.Scope) (domain.ScopeSummary, error) {
	items, err := s.ListEvidence(ctx, scope)
	if err != nil {
		return domain.ScopeSummary{}, err
	}
	batches, err := s.ListBatches(ctx, scope)
	if err != nil {
		return domain.ScopeSummary{}, err
	}
	return domain.SummarizeScope(scope, items, batches), nil
}
