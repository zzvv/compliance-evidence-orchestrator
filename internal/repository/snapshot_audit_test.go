package repository

import (
	"context"
	"testing"
	"time"

	"github.com/zzvv/compliance-evidence-orchestrator/internal/domain"
)

// TestSnapshotRestoreAuditIsolation 锁定 BUG_REPRO 中描述的快照/恢复缺陷回归：
// 恢复流程对快照内审计事件属性的临时编辑不得污染仓储中已保存的审计轨迹，
// 并且恢复后继续编辑旧快照也不得改动恢复后的记录。同时验证快照、恢复与
// 审计查询的正向行为保持可用。
func TestSnapshotRestoreAuditIsolation(t *testing.T) {
	store := NewStore()
	ctx := context.Background()
	scope := domain.Scope{ProjectID: "p", MaterialID: "m"}
	now := time.Now()

	source := domain.NewAuditEvent("a1", scope, "b1", "submitted", "alice", now)
	source.Attributes["source"] = "kept"
	if err := store.AppendAudit(ctx, source); err != nil {
		t.Fatal(err)
	}

	snap, err := store.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// 正向行为：快照应包含已保存的审计事件及其属性。
	snapEvents := snap.Audits[scope.Key()]
	if len(snapEvents) != 1 || snapEvents[0].Attributes["source"] != "kept" {
		t.Fatalf("snapshot missing audit event: %+v", snapEvents)
	}

	// 恢复流程对快照的临时编辑不得泄漏回仓储。
	snap.Audits[scope.Key()][0].Attributes["temp"] = "draft"
	if got, _ := store.ListAudit(ctx, scope); got[0].Attributes["temp"] != "" {
		t.Fatalf("store audit polluted by snapshot edit: %q", got[0].Attributes["temp"])
	}

	// 恢复后仓储应保留事件数据。
	if err := store.Restore(ctx, snap); err != nil {
		t.Fatal(err)
	}
	restored, err := store.ListAudit(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != 1 || restored[0].Attributes["source"] != "kept" || restored[0].Attributes["temp"] != "draft" {
		t.Fatalf("restored audit mismatch: %+v", restored)
	}

	// 恢复后继续编辑旧快照不得改动恢复后的审计记录。
	snap.Audits[scope.Key()][0].Attributes["late"] = "sneaky"
	if again, _ := store.ListAudit(ctx, scope); again[0].Attributes["late"] != "" {
		t.Fatalf("restored audit polluted by old snapshot edit: %q", again[0].Attributes["late"])
	}
}
