package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type BatchState string

const (
	BatchDraft       BatchState = "draft"
	BatchSubmitted   BatchState = "submitted"
	BatchUnderReview BatchState = "under_review"
	BatchApproved    BatchState = "approved"
	BatchRejected    BatchState = "rejected"
	BatchCancelled   BatchState = "cancelled"
)

type ReviewBatch struct {
	ID          string     `json:"id"`
	Scope       Scope      `json:"scope"`
	EvidenceIDs []string   `json:"evidence_ids"`
	State       BatchState `json:"state"`
	Reason      string     `json:"reason,omitempty"`
	Revision    int        `json:"revision"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func NewBatch(id string, scope Scope, evidenceIDs []string, now time.Time) (ReviewBatch, error) {
	batch := ReviewBatch{ID: strings.TrimSpace(id), Scope: scope, EvidenceIDs: normalizeIDs(evidenceIDs), State: BatchDraft, Revision: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC()}
	if err := batch.Validate(); err != nil {
		return ReviewBatch{}, err
	}
	return batch, nil
}
func (b ReviewBatch) Validate() error {
	if b.ID == "" {
		return fmt.Errorf("batch id is required")
	}
	if err := b.Scope.Validate(); err != nil {
		return err
	}
	if len(b.EvidenceIDs) == 0 {
		return fmt.Errorf("a review batch needs evidence")
	}
	if b.Revision < 1 {
		return fmt.Errorf("revision must be positive")
	}
	return nil
}
func (b ReviewBatch) CanTransition(to BatchState) bool {
	switch b.State {
	case BatchDraft:
		return to == BatchSubmitted || to == BatchCancelled
	case BatchSubmitted:
		return to == BatchUnderReview || to == BatchCancelled
	case BatchUnderReview:
		return to == BatchApproved || to == BatchRejected || to == BatchCancelled
	default:
		return false
	}
}
func (b *ReviewBatch) Transition(to BatchState, reason string, now time.Time) error {
	if !b.CanTransition(to) {
		return fmt.Errorf("batch %s cannot move from %s to %s", b.ID, b.State, to)
	}
	b.State = to
	b.Reason = strings.TrimSpace(reason)
	b.Revision++
	b.UpdatedAt = now.UTC()
	return nil
}
func (b ReviewBatch) IsTerminal() bool {
	return b.State == BatchApproved || b.State == BatchRejected || b.State == BatchCancelled
}
func normalizeIDs(values []string) []string {
	set := make(map[string]struct{})
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
