package worker

import "time"

type Backoff struct {
	Base    time.Duration
	Ceiling time.Duration
}

func DefaultBackoff() Backoff { return Backoff{Base: time.Second, Ceiling: 5 * time.Minute} }
func (b Backoff) Delay(attempt int) time.Duration {
	if attempt < 1 {
		return 0
	}
	delay := b.Base
	for index := 1; index < attempt && delay < b.Ceiling; index++ {
		delay *= 2
	}
	if delay > b.Ceiling {
		return b.Ceiling
	}
	return delay
}
