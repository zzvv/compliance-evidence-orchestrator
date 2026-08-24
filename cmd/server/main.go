package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/application"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/transport"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/worker"
)

func main() {
	store := repository.NewStore()
	notifier := application.NewMemoryNotifier()
	service := application.NewEvidenceService(store, store, store, notifier)
	server := &http.Server{Addr: address(), Handler: transport.NewRouter(service)}
	dispatcher := worker.NewDispatcher(service, 2*time.Second)
	// The reaper periodically archives batches whose retention window has
	// elapsed. Archival cascades to receipts, pending notifications and audit
	// events so the dispatcher and audit trail never surface dead batches.
	reaper := worker.NewReaper(func(ctx context.Context, now time.Time) error {
		_, err := service.SweepExpiredBatches(ctx, now)
		return err
	}, time.Minute)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go dispatcher.Run(ctx)
	go reaper.Run(ctx)
	go func() {
		log.Printf("compliance evidence orchestrator listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
}

func address() string {
	if value := os.Getenv("ADDR"); value != "" {
		return value
	}
	return ":8080"
}
