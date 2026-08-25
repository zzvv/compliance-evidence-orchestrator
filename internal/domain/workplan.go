package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// WorkActionKind describes the next operator action for a material scope.
type WorkActionKind string

const (
	WorkActionReplaceExpiredEvidence WorkActionKind = "replace_expired_evidence"
	WorkActionRenewExpiringEvidence  WorkActionKind = "renew_expiring_evidence"
	WorkActionStartReview            WorkActionKind = "start_review"
	WorkActionCompleteReview         WorkActionKind = "complete_review"
	WorkActionInvestigateRejection   WorkActionKind = "investigate_rejection"
)

const (
	WorkPriorityLow    = 1
	WorkPriorityNormal = 2
	WorkPriorityHigh   = 3
	WorkPriorityUrgent = 4
)

type WorkAction struct {
	Kind       WorkActionKind `json:"kind"`
	Priority   int            `json:"priority"`
	Scope      Scope          `json:"scope"`
	EvidenceID string         `json:"evidence_id,omitempty"`
	BatchID    string         `json:"batch_id,omitempty"`
	Reason     string         `json:"reason"`
	DueAt      time.Time      `json:"due_at"`
}

func (a WorkAction) Validate() error {
	if err := a.Scope.Validate(); err != nil {
		return err
	}
	if a.Priority < WorkPriorityLow || a.Priority > WorkPriorityUrgent {
		return fmt.Errorf("invalid work priority %d", a.Priority)
	}
	if strings.TrimSpace(a.Reason) == "" {
		return fmt.Errorf("work action reason is required")
	}
	if a.DueAt.IsZero() {
		return fmt.Errorf("work action due time is required")
	}
	switch a.Kind {
	case WorkActionReplaceExpiredEvidence, WorkActionRenewExpiringEvidence:
		if strings.TrimSpace(a.EvidenceID) == "" {
			return fmt.Errorf("evidence work action requires an evidence id")
		}
	case WorkActionStartReview, WorkActionCompleteReview, WorkActionInvestigateRejection:
		if strings.TrimSpace(a.BatchID) == "" {
			return fmt.Errorf("batch work action requires a batch id")
		}
	default:
		return fmt.Errorf("unsupported work action %q", a.Kind)
	}
	return nil
}

type WorkPlan struct {
	Scope       Scope        `json:"scope"`
	GeneratedAt time.Time    `json:"generated_at"`
	Actions     []WorkAction `json:"actions"`
	Evidence    EvidenceWork `json:"evidence"`
	Batches     BatchWork    `json:"batches"`
}

type EvidenceWork struct {
	Total      int `json:"total"`
	Expired    int `json:"expired"`
	Expiring   int `json:"expiring"`
	WithoutTTL int `json:"without_ttl"`
}

type BatchWork struct {
	Submitted   int `json:"submitted"`
	UnderReview int `json:"under_review"`
	Rejected    int `json:"rejected"`
	Completed   int `json:"completed"`
}

func (p WorkPlan) HasUrgentAction() bool {
	return len(p.Actions) > 0 && p.Actions[0].Priority == WorkPriorityUrgent
}

func (p WorkPlan) NextAction() (WorkAction, bool) {
	if len(p.Actions) == 0 {
		return WorkAction{}, false
	}
	return p.Actions[0], true
}

func (p WorkPlan) ActionsForBatch(batchID string) []WorkAction {
	batchID = strings.TrimSpace(batchID)
	result := make([]WorkAction, 0)
	for _, action := range p.Actions {
		if action.BatchID == batchID {
			result = append(result, action)
		}
	}
	return result
}

func (p WorkPlan) CountByPriority() map[int]int {
	counts := map[int]int{
		WorkPriorityLow:    0,
		WorkPriorityNormal: 0,
		WorkPriorityHigh:   0,
		WorkPriorityUrgent: 0,
	}
	for _, action := range p.Actions {
		counts[action.Priority]++
	}
	return counts
}

// BuildWorkPlan converts the current scope state into an ordered operator queue.
func BuildWorkPlan(scope Scope, evidence []Evidence, batches []ReviewBatch, now time.Time) (WorkPlan, error) {
	if err := scope.Validate(); err != nil {
		return WorkPlan{}, err
	}
	now = now.UTC()
	plan := WorkPlan{
		Scope:       scope,
		GeneratedAt: now,
		Actions:     make([]WorkAction, 0),
	}
	for _, item := range evidence {
		if item.Scope() != scope {
			return WorkPlan{}, fmt.Errorf("evidence %s is outside work plan scope", item.ID)
		}
		plan.Evidence.Total++
		actions := evidenceActions(scope, item, now, &plan.Evidence)
		plan.Actions = append(plan.Actions, actions...)
	}
	for _, batch := range batches {
		if batch.Scope != scope {
			return WorkPlan{}, fmt.Errorf("batch %s is outside work plan scope", batch.ID)
		}
		actions := batchActions(scope, batch, now, &plan.Batches)
		plan.Actions = append(plan.Actions, actions...)
	}
	sortWorkActions(plan.Actions)
	return plan, nil
}

func evidenceActions(scope Scope, item Evidence, now time.Time, summary *EvidenceWork) []WorkAction {
	if item.ExpiresAt == nil {
		summary.WithoutTTL++
		return nil
	}
	expiresAt := item.ExpiresAt.UTC()
	if !expiresAt.After(now) {
		summary.Expired++
		return []WorkAction{{
			Kind:       WorkActionReplaceExpiredEvidence,
			Priority:   WorkPriorityUrgent,
			Scope:      scope,
			EvidenceID: item.ID,
			Reason:     fmt.Sprintf("证据 %s 已过期", item.Reference),
			DueAt:      expiresAt,
		}}
	}
	if expiresAt.Sub(now) <= 30*24*time.Hour {
		summary.Expiring++
		return []WorkAction{{
			Kind:       WorkActionRenewExpiringEvidence,
			Priority:   WorkPriorityHigh,
			Scope:      scope,
			EvidenceID: item.ID,
			Reason:     fmt.Sprintf("证据 %s 将在 30 天内到期", item.Reference),
			DueAt:      expiresAt,
		}}
	}
	return nil
}

func batchActions(scope Scope, batch ReviewBatch, now time.Time, summary *BatchWork) []WorkAction {
	switch batch.State {
	case BatchSubmitted:
		summary.Submitted++
		return []WorkAction{{
			Kind:     WorkActionStartReview,
			Priority: WorkPriorityHigh,
			Scope:    scope,
			BatchID:  batch.ID,
			Reason:   "批次等待审核人员开始处理",
			DueAt:    dueFrom(batch.UpdatedAt, now, 24*time.Hour),
		}}
	case BatchUnderReview:
		summary.UnderReview++
		return []WorkAction{{
			Kind:     WorkActionCompleteReview,
			Priority: WorkPriorityHigh,
			Scope:    scope,
			BatchID:  batch.ID,
			Reason:   "批次正在审核，需记录最终决定",
			DueAt:    dueFrom(batch.UpdatedAt, now, 48*time.Hour),
		}}
	case BatchRejected:
		summary.Rejected++
		return []WorkAction{{
			Kind:     WorkActionInvestigateRejection,
			Priority: WorkPriorityNormal,
			Scope:    scope,
			BatchID:  batch.ID,
			Reason:   "批次已驳回，需跟进材料整改",
			DueAt:    dueFrom(batch.UpdatedAt, now, 7*24*time.Hour),
		}}
	case BatchApproved, BatchCancelled:
		summary.Completed++
	}
	return nil
}

func dueFrom(updatedAt, now time.Time, target time.Duration) time.Time {
	due := updatedAt.UTC().Add(target)
	if due.Before(now) {
		return now
	}
	return due
}

func sortWorkActions(actions []WorkAction) {
	sort.SliceStable(actions, func(left, right int) bool {
		if actions[left].Priority != actions[right].Priority {
			return actions[left].Priority > actions[right].Priority
		}
		if !actions[left].DueAt.Equal(actions[right].DueAt) {
			return actions[left].DueAt.Before(actions[right].DueAt)
		}
		if actions[left].BatchID != actions[right].BatchID {
			return actions[left].BatchID < actions[right].BatchID
		}
		return actions[left].EvidenceID < actions[right].EvidenceID
	})
}
