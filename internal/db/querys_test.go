package db

import (
	"testing"

	"quick-feishu/internal/model"
)

func TestSnapshotQueries(t *testing.T) {
	dir := t.TempDir()
	gdb, _ := Init(dir)
	gdb.Create(&model.Snapshot{SnapshotDate: "2026-09-17", AccountUsed: 100})
	gdb.Create(&model.Snapshot{SnapshotDate: "2026-09-18", AccountUsed: 150})

	latest, err := LatestSnapshot(gdb)
	if err != nil {
		t.Fatal(err)
	}
	if latest.SnapshotDate != "2026-09-18" {
		t.Errorf("latest = %s", latest.SnapshotDate)
	}
	before, err := SnapshotBefore(gdb, "2026-09-18")
	if err != nil {
		t.Fatal(err)
	}
	if before.AccountUsed != 150 {
		t.Errorf("before 09-18 should be 150, got %d", before.AccountUsed)
	}
	before2, err := SnapshotBefore(gdb, "2026-09-17")
	if err != nil {
		t.Fatal(err)
	}
	if before2.AccountUsed != 100 {
		t.Errorf("before 09-17 should be 100, got %d", before2.AccountUsed)
	}
	// 09-16 之前无快照：返回错误（"向前替代"由调用方处理，见 Task 16）
	if _, err := SnapshotBefore(gdb, "2026-09-16"); err == nil {
		t.Errorf("expected error when no snapshot at or before 09-16")
	}
	inRange, err := SnapshotsInRange(gdb, "2026-09-17", "2026-09-18")
	if err != nil {
		t.Fatal(err)
	}
	if len(inRange) != 2 {
		t.Errorf("range = %d, want 2", len(inRange))
	}
	// 分页查询
	list, total, err := ListSnapshots(gdb, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(list) != 2 {
		t.Errorf("list total=%d len=%d, want 2/2", total, len(list))
	}
	if list[0].SnapshotDate != "2026-09-18" {
		t.Errorf("list should be desc order, got first=%s", list[0].SnapshotDate)
	}
	// 按日期查询
	byDate, err := SnapshotByDate(gdb, "2026-09-17")
	if err != nil {
		t.Fatal(err)
	}
	if byDate.AccountUsed != 100 {
		t.Errorf("byDate used = %d", byDate.AccountUsed)
	}
	// 令牌查询
	ts, _ := TokenSnapshots(gdb, latest.ID)
	if len(ts) != 0 {
		t.Errorf("tokens for empty snapshot = %d, want 0", len(ts))
	}
	// 发送日志分页
	gdb.Create(&model.SendLog{Date: "2026-09-18", Success: true})
	logs, ltotal, err := ListSendLogs(gdb, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if ltotal != 1 || len(logs) != 1 {
		t.Errorf("logs total=%d len=%d, want 1/1", ltotal, len(logs))
	}
}

func TestLatestTokensFallsBackPastAccountOnlySnapshot(t *testing.T) {
	gdb, err := Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	withTokens := &model.Snapshot{SnapshotDate: "2026-09-20"}
	if err := gdb.Create(withTokens).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Create(&model.TokenSnapshot{SnapshotID: withTokens.ID, TokenID: 7, TokenName: "saved-token"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Create(&model.Snapshot{SnapshotDate: "2026-09-21"}).Error; err != nil {
		t.Fatal(err)
	}

	tokens, err := LatestTokens(gdb)
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) != 1 || tokens[0].TokenName != "saved-token" {
		t.Fatalf("latest tokens = %+v, want fallback token", tokens)
	}
}
