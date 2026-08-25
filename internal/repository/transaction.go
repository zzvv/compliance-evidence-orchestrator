package repository

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"sort"
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
		snapshot.Evidence = append(snapshot.Evidence, value)
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
		s.evidence[value.Scope().Key()+":"+value.ID] = value
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

// ScopeView bundles the evidence and batches that belong to one scope so that
// callers (such as the review dashboard) read a single consistent snapshot
// instead of stitching together values observed across concurrent writes.
type ScopeView struct {
	Evidence []domain.Evidence
	Batches  []domain.ReviewBatch
}

// ScopeViewReader returns a scope-scoped consistent snapshot of evidence and
// batches taken under one read lock. Implementations must produce results that
// are internally consistent at a single point in time.
type ScopeViewReader interface {
	ReadScopeView(context.Context, domain.Scope) (ScopeView, error)
}

// ReadScopeView returns all evidence and batches for scope under a single read
// lock, guaranteeing the two collections were observed at the same instant.
// Sorting mirrors the standalone read paths: evidence by ID ascending and
// batches by CreatedAt ascending, matching ListEvidence and ListBatches.
func (s *Store) ReadScopeView(ctx context.Context, scope domain.Scope) (ScopeView, error) {
	if err := ctx.Err(); err != nil {
		return ScopeView{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	evidence := make([]domain.Evidence, 0)
	for _, item := range s.evidence {
		if item.Scope() == scope {
			evidence = append(evidence, item)
		}
	}
	sort.Slice(evidence, func(i, j int) bool { return evidence[i].ID < evidence[j].ID })
	batches := make([]domain.ReviewBatch, 0)
	for _, item := range s.batches {
		if item.Scope == scope {
			batches = append(batches, copyBatch(item))
		}
	}
	sort.Slice(batches, func(i, j int) bool { return batches[i].CreatedAt.Before(batches[j].CreatedAt) })
	return ScopeView{Evidence: evidence, Batches: batches}, nil
}
