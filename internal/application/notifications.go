package application

import (
	"context"
	"errors"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

func (s *EvidenceService) queueNotification(ctx context.Context, batch domain.ReviewBatch, recipient, event string) {
	if s.notifications == nil {
		return
	}
	now := time.Now()
	notification := domain.Notification{ID: s.ids.New("notice"), BatchID: batch.ID, Recipient: recipient, Event: event, State: domain.NotificationPending, CreatedAt: now, UpdatedAt: now}
	_ = s.notifications.SaveNotification(ctx, notification)
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
		// 投递前必须先原子地领取该通知。两个调度实例可能同时读到同一份
		// 待发送列表，只有领取成功的一方才有权投递；另一方会发现状态已
		// 变为 dispatching 或 delivered 而跳过，避免重复投递。
		claimed, err := s.notifications.ClaimNotification(ctx, item.ID)
		if err != nil {
			if errors.Is(err, domain.ErrConflict) || errors.Is(err, domain.ErrNotFound) {
				continue
			}
			return err
		}
		if err := s.notifier.Deliver(ctx, claimed.Recipient, claimed.Event); err != nil {
			claimed.MarkFailed(err, time.Now())
		} else {
			claimed.MarkDelivered(time.Now())
		}
		if err := s.notifications.SaveNotification(ctx, claimed); err != nil {
			return err
		}
	}
	return nil
}
