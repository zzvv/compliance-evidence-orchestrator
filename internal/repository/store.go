package repository

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"sort"
	"sync"
	"time"
)

type Store struct {
	mu            sync.RWMutex
	evidence      map[string]domain.Evidence
	batches       map[string]domain.ReviewBatch
	receipts      map[string][]domain.Receipt
	notifications map[string]domain.Notification
	audits        map[string][]domain.AuditEvent
	retention     domain.RetentionRule
}

func NewStore() *Store {
	return &Store{evidence: map[string]domain.Evidence{}, batches: map[string]domain.ReviewBatch{}, receipts: map[string][]domain.Receipt{}, notifications: map[string]domain.Notification{}, audits: map[string][]domain.AuditEvent{}, retention: domain.DefaultRetentionRule()}
}

// WithRetentionRule overrides the retention policy used when sweeping expired
// batches. Tests use it to shrink the keep windows so archival can be exercised
// without waiting days.
func (s *Store) WithRetentionRule(rule domain.RetentionRule) *Store {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retention = rule
	return s
}

// ListExpiredBatches returns terminal batches whose retention window has elapsed
// as of now. Used by the application sweep to drive batch archival.
func (s *Store) ListExpiredBatches(ctx context.Context, now time.Time) ([]domain.ReviewBatch, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.ReviewBatch, 0)
	for _, batch := range s.batches {
		if s.retention.IsExpired(batch, now) {
			result = append(result, copyBatch(batch))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.Before(result[j].UpdatedAt) })
	return result, nil
}
func (s *Store) SaveEvidence(ctx context.Context, e domain.Evidence) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evidence[e.Scope().Key()+":"+e.ID] = e
	return nil
}
func (s *Store) ListEvidence(ctx context.Context, scope domain.Scope) ([]domain.Evidence, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []domain.Evidence{}
	for _, item := range s.evidence {
		if item.Scope() == scope {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
func (s *Store) FindEvidence(ctx context.Context, scope domain.Scope, id string) (domain.Evidence, error) {
	if err := ctx.Err(); err != nil {
		return domain.Evidence{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.evidence[scope.Key()+":"+id]
	if !ok {
		return domain.Evidence{}, domain.ErrNotFound
	}
	return item, nil
}
func (s *Store) SaveBatch(ctx context.Context, b domain.ReviewBatch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.batches[b.ID]
	if ok && current.Revision >= b.Revision {
		return domain.ErrConflict
	}
	s.batches[b.ID] = copyBatch(b)
	return nil
}
func (s *Store) FindBatch(ctx context.Context, id string) (domain.ReviewBatch, error) {
	if err := ctx.Err(); err != nil {
		return domain.ReviewBatch{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.batches[id]
	if !ok {
		return domain.ReviewBatch{}, domain.ErrNotFound
	}
	return copyBatch(item), nil
}
func (s *Store) ListBatches(ctx context.Context, scope domain.Scope) ([]domain.ReviewBatch, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []domain.ReviewBatch{}
	for _, item := range s.batches {
		if item.Scope == scope {
			result = append(result, copyBatch(item))
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result, nil
}
func (s *Store) AppendReceipt(ctx context.Context, r domain.Receipt) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.receipts[r.BatchID] = append(s.receipts[r.BatchID], r)
	return nil
}
func (s *Store) ListReceipts(ctx context.Context, batchID string) ([]domain.Receipt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.Receipt(nil), s.receipts[batchID]...), nil
}
func (s *Store) SaveNotification(ctx context.Context, n domain.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifications[n.ID] = n
	return nil
}
func (s *Store) PendingNotifications(ctx context.Context, limit int) ([]domain.Notification, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []domain.Notification{}
	for _, n := range s.notifications {
		if n.State == domain.NotificationPending || n.State == domain.NotificationFailed {
			result = append(result, n)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
func (s *Store) AppendAudit(ctx context.Context, event domain.AuditEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := event.Scope.Key()
	s.audits[key] = append(s.audits[key], event)
	return nil
}
func (s *Store) ListAudit(ctx context.Context, scope domain.Scope) ([]domain.AuditEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.AuditEvent(nil), s.audits[scope.Key()]...), nil
}
func copyBatch(batch domain.ReviewBatch) domain.ReviewBatch {
	batch.EvidenceIDs = append([]string(nil), batch.EvidenceIDs...)
	return batch
}
