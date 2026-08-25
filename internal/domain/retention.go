package domain

import "time"

type RetentionRule struct {
	KeepApprovedFor  time.Duration
	KeepRejectedFor  time.Duration
	KeepCancelledFor time.Duration
}

func DefaultRetentionRule() RetentionRule {
	return RetentionRule{KeepApprovedFor: 365 * 24 * time.Hour, KeepRejectedFor: 180 * 24 * time.Hour, KeepCancelledFor: 30 * 24 * time.Hour}
}
func (r RetentionRule) DeleteAfter(batch ReviewBatch) time.Time {
	duration := r.KeepCancelledFor
	switch batch.State {
	case BatchApproved:
		duration = r.KeepApprovedFor
	case BatchRejected:
		duration = r.KeepRejectedFor
	}
	return batch.UpdatedAt.Add(duration)
}
func (r RetentionRule) IsExpired(batch ReviewBatch, now time.Time) bool {
	return batch.IsTerminal() && !r.DeleteAfter(batch).After(now)
}
