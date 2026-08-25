package domain

import (
	"sort"
	"time"
)

type TimelineEntry struct {
	At      time.Time `json:"at"`
	Kind    string    `json:"kind"`
	Message string    `json:"message"`
	Actor   string    `json:"actor,omitempty"`
}

func BuildTimeline(batch ReviewBatch, receipts []Receipt, audit []AuditEvent) []TimelineEntry {
	entries := make([]TimelineEntry, 0, len(receipts)+len(audit)+1)
	entries = append(entries, TimelineEntry{At: batch.CreatedAt, Kind: "batch_created", Message: "审核批次已创建"})
	for _, receipt := range receipts {
		entries = append(entries, TimelineEntry{At: receipt.RecordedAt, Kind: string(receipt.Kind), Message: receipt.Message})
	}
	for _, event := range audit {
		if event.BatchID == batch.ID {
			entries = append(entries, TimelineEntry{At: event.At, Kind: event.Action, Message: event.Action, Actor: event.Actor})
		}
	}
	sort.SliceStable(entries, func(left, right int) bool { return entries[left].At.Before(entries[right].At) })
	return entries
}

func LatestTimelineEntry(entries []TimelineEntry) (TimelineEntry, bool) {
	if len(entries) == 0 {
		return TimelineEntry{}, false
	}
	return entries[len(entries)-1], true
}
func TimelineSince(entries []TimelineEntry, at time.Time) []TimelineEntry {
	result := make([]TimelineEntry, 0)
	for _, entry := range entries {
		if !entry.At.Before(at) {
			result = append(result, entry)
		}
	}
	return result
}
