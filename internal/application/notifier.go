package application

import (
	"context"
	"sync"
)

type Notifier interface {
	Deliver(context.Context, string, string) error
}
type Delivery struct {
	Recipient string
	Event     string
}
type MemoryNotifier struct {
	mu         sync.Mutex
	deliveries []Delivery
	failure    error
}

func NewMemoryNotifier() *MemoryNotifier { return &MemoryNotifier{} }
func (n *MemoryNotifier) Deliver(ctx context.Context, recipient, event string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.failure != nil {
		return n.failure
	}
	n.deliveries = append(n.deliveries, Delivery{Recipient: recipient, Event: event})
	return nil
}
func (n *MemoryNotifier) SetFailure(err error) { n.mu.Lock(); defer n.mu.Unlock(); n.failure = err }
func (n *MemoryNotifier) Deliveries() []Delivery {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]Delivery(nil), n.deliveries...)
}
