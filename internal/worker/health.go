package worker

import (
	"sync"
	"time"
)

type Health struct {
	mu        sync.RWMutex
	lastRun   time.Time
	lastError error
}

func (h *Health) Record(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastRun = time.Now().UTC()
	h.lastError = err
}
func (h *Health) LastRun() time.Time { h.mu.RLock(); defer h.mu.RUnlock(); return h.lastRun }
func (h *Health) LastError() error   { h.mu.RLock(); defer h.mu.RUnlock(); return h.lastError }
