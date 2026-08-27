package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/application"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
	"github.com/zzvv/compliance-evidence-orchestrator/internal/repository"
)

func TestScopeSummaryRouteReturnsCurrentScopeCounts(t *testing.T) {
	store := repository.NewStore()
	service := application.NewEvidenceService(store, store, store, application.NewMemoryNotifier())
	ctx := context.Background()
	evidence, err := service.RegisterEvidence(ctx, application.RegisterEvidenceCommand{
		ProjectID: "project-a", MaterialID: "material-a", Reference: "DOC-1", Kind: domain.Certificate,
		Supplier: "supplier", IssuedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateBatch(ctx, application.CreateBatchCommand{
		ProjectID: "project-a", MaterialID: "material-a", EvidenceIDs: []string{evidence.ID}, Actor: "operator",
	}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/scopes/project-a/material-a/summary", nil)
	response := httptest.NewRecorder()
	NewRouter(service).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Body.String(); got == "" || !contains(got, `"total_evidence":1`) || !contains(got, `"open_batches":1`) {
		t.Fatalf("summary response = %s", got)
	}
}

func contains(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
