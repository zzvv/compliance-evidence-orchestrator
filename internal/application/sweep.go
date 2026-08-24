package application

import (
	"context"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

// BatchArchiver is the repository capability needed to sweep expired batches.
// It is satisfied by the in-memory Store and any persistent implementation that
// wants the same archival behaviour.
type BatchArchiver interface {
	ListExpiredBatches(ctx context.Context, now time.Time) ([]domain.ReviewBatch, error)
	DeleteBatch(ctx context.Context, batchID string) error
}

// SweepResult records what an archival sweep removed so callers (the reaper,
// operational dashboards) can report on the cleanup deterministically.
type SweepResult struct {
	ArchivedBatchIDs []string `json:"archived_batch_ids"`
}

// SweepExpiredBatches removes every terminal batch whose retention window has
// elapsed as of now. Each removal cascades to the batch's receipts, pending
// notifications and audit events (see Store.DeleteBatch), so an archived batch
// leaves no trace that the dispatcher or audit trail could resurrect. The sweep
// is best-effort: a failed batch deletion is recorded but does not abort the
// remaining sweeps, and already-cleaned batches simply contribute nothing.
func (s *EvidenceService) SweepExpiredBatches(ctx context.Context, now time.Time) (SweepResult, error) {
	archiver, ok := s.batches.(BatchArchiver)
	if !ok {
		return SweepResult{}, nil
	}
	expired, err := archiver.ListExpiredBatches(ctx, now)
	if err != nil {
		return SweepResult{}, err
	}
	result := SweepResult{ArchivedBatchIDs: make([]string, 0, len(expired))}
	for _, batch := range expired {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if err := archiver.DeleteBatch(ctx, batch.ID); err != nil {
			continue
		}
		result.ArchivedBatchIDs = append(result.ArchivedBatchIDs, batch.ID)
	}
	return result, nil
}
