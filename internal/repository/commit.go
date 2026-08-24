package repository

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

// SubmissionCommitter makes the batch and its initial receipt visible together.
type SubmissionCommitter interface {
	CommitSubmission(context.Context, domain.ReviewBatch, domain.Receipt) error
}

func (s *Store) CommitSubmission(ctx context.Context, batch domain.ReviewBatch, receipt domain.Receipt) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.batches[batch.ID]; exists {
		return domain.ErrConflict
	}
	if receipt.BatchID != batch.ID {
		return domain.ErrConflict
	}
	s.batches[batch.ID] = copyBatch(batch)
	s.receipts[batch.ID] = append(s.receipts[batch.ID], receipt)
	return nil
}
func (s *Store) DeleteBatch(ctx context.Context, batchID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.batches[batchID]; !ok {
		return domain.ErrNotFound
	}
	delete(s.batches, batchID)
	delete(s.receipts, batchID)
	return nil
}
