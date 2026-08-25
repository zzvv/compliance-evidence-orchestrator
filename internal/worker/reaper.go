package worker

import (
	"context"
	"time"
)

type SweepFunc func(context.Context, time.Time) error
type Reaper struct {
	sweep    SweepFunc
	interval time.Duration
}

func NewReaper(sweep SweepFunc, interval time.Duration) *Reaper {
	return &Reaper{sweep: sweep, interval: interval}
}
func (r *Reaper) Run(ctx context.Context) {
	if r.sweep == nil {
		return
	}
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			_ = r.sweep(ctx, now)
		}
	}
}
