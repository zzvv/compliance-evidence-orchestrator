package application

import (
	"context"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"time"
)

func (s *EvidenceService) queueNotification(ctx context.Context, batch domain.ReviewBatch, recipient, event string) error {
	if s.notifications == nil {
		return nil
	}
	now := time.Now()
	notification := domain.Notification{ID: s.ids.New("notice"), BatchID: batch.ID, Recipient: recipient, Event: event, State: domain.NotificationPending, CreatedAt: now, UpdatedAt: now}
	return s.notifications.SaveNotification(ctx, notification)
}
func (s *EvidenceService) DispatchPending(ctx context.Context, limit int) error {
	if s.notifications == nil {
		return nil
	}
	items, err := s.notifications.PendingNotifications(ctx, limit)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.notifier.Deliver(ctx, item.Recipient, item.Event); err != nil {
			item.MarkFailed(err, time.Now())
		} else {
			item.MarkDelivered(time.Now())
		}
		// Delivery is an external side effect. Once it returns, preserve its
		// outcome even when the dispatcher was cancelled in the meantime.
		if err := s.notifications.SaveNotification(context.WithoutCancel(ctx), item); err != nil {
			return err
		}
	}
	return nil
}
