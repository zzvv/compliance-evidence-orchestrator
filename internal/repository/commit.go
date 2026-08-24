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
	batch, ok := s.batches[batchID]
	if !ok {
		return domain.ErrNotFound
	}
	delete(s.batches, batchID)
	delete(s.receipts, batchID)
	s.purgeNotificationsLocked(batchID)
	s.purgeAuditLocked(batch.Scope, batchID)
	return nil
}

// PurgeNotificationsForBatch drops every notification tied to batchID. The
// dispatcher only re-issues pending notifications, but delivered/failed records
// are dropped too so an archived batch leaves nothing behind for export.
func (s *Store) PurgeNotificationsForBatch(ctx context.Context, batchID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeNotificationsLocked(batchID)
	return nil
}

// PurgeAuditForBatch removes audit events that reference batchID. Events are
// partitioned by scope, so the sibling batches within the same scope are kept.
func (s *Store) PurgeAuditForBatch(ctx context.Context, batchID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for scopeKey, events := range s.audits {
		filtered := s.audits[scopeKey][:0]
		changed := false
		for _, event := range events {
			if event.BatchID == batchID {
				changed = true
				continue
			}
			filtered = append(filtered, event)
		}
		if changed {
			s.audits[scopeKey] = filtered
		}
	}
	return nil
}

func (s *Store) purgeNotificationsLocked(batchID string) {
	for id, notification := range s.notifications {
		if notification.BatchID == batchID {
			delete(s.notifications, id)
		}
	}
}

func (s *Store) purgeAuditLocked(scope domain.Scope, batchID string) {
	events, ok := s.audits[scope.Key()]
	if !ok {
		return
	}
	filtered := s.audits[scope.Key()][:0]
	changed := false
	for _, event := range events {
		if event.BatchID == batchID {
			changed = true
			continue
		}
		filtered = append(filtered, event)
	}
	if changed {
		s.audits[scope.Key()] = filtered
	}
}
