package repository

import (
	"context"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

func TestSnapshotAuditAttributesCannotMutateStoredTimeline(t *testing.T) {
	store := NewStore()
	ctx := context.Background()
	scope, err := domain.NewScope("project-a", "material-a")
	if err != nil {
		t.Fatal(err)
	}
	event := domain.NewAuditEvent("audit-1", scope, "batch-1", "review_started", "reviewer", time.Now())
	event.Attributes["source"] = "workflow"
	if err := store.AppendAudit(ctx, event); err != nil {
		t.Fatal(err)
	}

	snapshot, err := store.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Audits[scope.Key()][0].Attributes["replay_note"] = "temporary"

	audit, err := store.ListAudit(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := audit[0].Attributes["replay_note"]; found {
		t.Fatal("mutating a recovery snapshot changed the stored audit trail")
	}
}

func TestRestoreDoesNotRetainSnapshotAuditAttributeAliases(t *testing.T) {
	store := NewStore()
	ctx := context.Background()
	scope, err := domain.NewScope("project-a", "material-a")
	if err != nil {
		t.Fatal(err)
	}
	event := domain.NewAuditEvent("audit-1", scope, "batch-1", "review_started", "reviewer", time.Now())
	event.Attributes["source"] = "workflow"
	if err := store.AppendAudit(ctx, event); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Restore(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	snapshot.Audits[scope.Key()][0].Attributes["restored_note"] = "temporary"

	audit, err := store.ListAudit(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := audit[0].Attributes["restored_note"]; found {
		t.Fatal("mutating a consumed snapshot changed the restored audit trail")
	}
}
