package application

import (
	"fmt"
	"sync/atomic"
	"time"
)

type IDGenerator struct{ sequence atomic.Uint64 }

func (g *IDGenerator) New(prefix string) string {
	return fmt.Sprintf("%s-%d-%04d", prefix, time.Now().UTC().UnixNano(), g.sequence.Add(1))
}
