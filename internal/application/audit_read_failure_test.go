package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

var errAuditStoreUnavailable = errors.New("audit store unavailable")

type auditReadFailureStore struct{ *repository.Store }

func (s *auditReadFailureStore) ListAudit(context.Context, domain.Scope) ([]domain.AuditEvent, error) {
	return nil, errAuditStoreUnavailable
}

func TestAuditReadFailureCannotMasqueradeAsCompleteBatchHistory(t *testing.T) {
	store := &auditReadFailureStore{Store: repository.NewStore()}
	service := NewEvidenceService(store, store, store, NewMemoryNotifier())
	ctx := context.Background()

	evidence, err := service.RegisterEvidence(ctx, RegisterEvidenceCommand{
		ProjectID: "project-a",
		MaterialID: "material-a",
		Reference: "CERT-01",
		Kind: domain.Certificate,
		Supplier: "supplier",
		IssuedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := service.CreateBatch(ctx, CreateBatchCommand{
		ProjectID: "project-a",
		MaterialID: "material-a",
		EvidenceIDs: []string{evidence.ID},
		Actor: "operator",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, detailErr := service.BatchDetail(ctx, batch.ID)
	_, timelineErr := service.Timeline(ctx, batch.ID)
	if !errors.Is(detailErr, errAuditStoreUnavailable) || !errors.Is(timelineErr, errAuditStoreUnavailable) {
		t.Fatalf("audit read failure was hidden: detail=%v timeline=%v", detailErr, timelineErr)
	}
}
