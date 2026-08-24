package repository

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

type Snapshot struct {
	Evidence      []domain.Evidence
	Batches       []domain.ReviewBatch
	Receipts      map[string][]domain.Receipt
	Notifications []domain.Notification
	Audits        map[string][]domain.AuditEvent
}

func (s *Store) Snapshot(ctx context.Context) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := Snapshot{Receipts: map[string][]domain.Receipt{}, Audits: map[string][]domain.AuditEvent{}}
	for _, value := range s.evidence {
		snapshot.Evidence = append(snapshot.Evidence, copyEvidence(value))
	}
	for _, value := range s.batches {
		snapshot.Batches = append(snapshot.Batches, copyBatch(value))
	}
	for key, values := range s.receipts {
		snapshot.Receipts[key] = append([]domain.Receipt(nil), values...)
	}
	for _, value := range s.notifications {
		snapshot.Notifications = append(snapshot.Notifications, value)
	}
	for key, values := range s.audits {
		snapshot.Audits[key] = append([]domain.AuditEvent(nil), values...)
	}
	return snapshot, nil
}
func (s *Store) Restore(ctx context.Context, snapshot Snapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evidence = map[string]domain.Evidence{}
	s.batches = map[string]domain.ReviewBatch{}
	s.receipts = map[string][]domain.Receipt{}
	s.notifications = map[string]domain.Notification{}
	s.audits = map[string][]domain.AuditEvent{}
	for _, value := range snapshot.Evidence {
		s.evidence[value.Scope().Key()+":"+value.ID] = copyEvidence(value)
	}
	for _, value := range snapshot.Batches {
		s.batches[value.ID] = copyBatch(value)
	}
	for key, values := range snapshot.Receipts {
		s.receipts[key] = append([]domain.Receipt(nil), values...)
	}
	for _, value := range snapshot.Notifications {
		s.notifications[value.ID] = value
	}
	for key, values := range snapshot.Audits {
		s.audits[key] = append([]domain.AuditEvent(nil), values...)
	}
	return nil
}
