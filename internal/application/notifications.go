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
		// 取消发生在开始投递前：保留待发送状态以便下一轮重试，且不再处理后续项。
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.notifier.Deliver(ctx, item.Recipient, item.Event); err != nil {
			// 投递返回错误时区分取消与真实失败：取消意味着投递未可靠完成，
			// 同样保留待发送状态并停止处理后续项；真实失败才记录为可重试的失败。
			if cerr := ctx.Err(); cerr != nil {
				return cerr
			}
			item.MarkFailed(err, time.Now())
		} else {
			item.MarkDelivered(time.Now())
		}
		// 投递的副作用已经发生，送达结果必须可靠落库：即便调度上下文此刻被取消，
		// 也使用不可取消上下文完成回写，避免下一轮对同一条通知重复投递。
		if err := s.notifications.SaveNotification(context.Background(), item); err != nil {
			return err
		}
	}
	return nil
}
