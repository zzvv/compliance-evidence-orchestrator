package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

var errNotificationStoreUnavailable = errors.New("notification store unavailable")

type notificationSaveFailureStore struct{ *repository.Store }

func (s *notificationSaveFailureStore) SaveNotification(context.Context, domain.Notification) error {
	return errNotificationStoreUnavailable
}

func TestDecisionDoesNotPretendSuccessWhenNotificationEnqueueFails(t *testing.T) {
	store := &notificationSaveFailureStore{Store: repository.NewStore()}
	service := NewEvidenceService(store, store, store, NewMemoryNotifier())
	ctx := context.Background()
	evidence, err := service.RegisterEvidence(ctx, RegisterEvidenceCommand{
		ProjectID: "project-a", MaterialID: "material-a", Reference: "CERT-01",
		Kind: domain.Certificate, Supplier: "supplier", IssuedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := service.CreateBatch(ctx, CreateBatchCommand{
		ProjectID: "project-a", MaterialID: "material-a", EvidenceIDs: []string{evidence.ID}, Actor: "operator",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartReview(ctx, batch.ID, "reviewer"); err != nil {
		t.Fatal(err)
	}

	_, err = service.DecideBatch(ctx, DecideBatchCommand{
		BatchID: batch.ID, Approved: true, Actor: "reviewer", Recipient: "supplier@example.test",
	})
	if !errors.Is(err, errNotificationStoreUnavailable) {
		t.Fatalf("notification enqueue failure was hidden: %v", err)
	}
}
