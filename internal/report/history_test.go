package report

import (
	"testing"
	"time"

	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
)

func TestBuildAccountHistory(t *testing.T) {
	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SeedDicts(gdb); err != nil {
		t.Fatal(err)
	}
	gdb.Create(&model.Snapshot{
		SnapshotDate: "2026-09-17",
		AccountRaw:   mustJSON(t, map[string]interface{}{"used_quota": float64(100), "username": "a"}),
	})
	gdb.Create(&model.Snapshot{
		SnapshotDate: "2026-09-18",
		AccountRaw:   mustJSON(t, map[string]interface{}{"used_quota": float64(150), "username": "a", "extra_field": "x"}),
	})

	h := BuildAccountHistory(gdb, 30)
	if h.Source != "account" {
		t.Errorf("source = %s", h.Source)
	}
	if len(h.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(h.Rows))
	}
	if h.Rows[0].Date != "2026-09-17" || h.Rows[1].Date != "2026-09-18" {
		t.Errorf("rows not ascending: %+v", h.Rows)
	}
	// 变动：09-18 相对 09-17 = +50
	if h.Rows[1].Deltas["used_quota"] != "+50" {
		t.Errorf("delta = %q, want +50", h.Rows[1].Deltas["used_quota"])
	}
	if h.Rows[0].Deltas["used_quota"] != "" {
		t.Errorf("first row should have no delta, got %q", h.Rows[0].Deltas["used_quota"])
	}
	// 字典字段带中文名
	var found bool
	for _, f := range h.Fields {
		if f.Path == "used_quota" && f.Label == "已用配额" {
			found = true
		}
	}
	if !found {
		t.Errorf("used_quota field with label missing: %+v", h.Fields)
	}
	// 额外字段也应出现
	var extra bool
	for _, f := range h.Fields {
		if f.Path == "extra_field" {
			extra = true
		}
	}
	if !extra {
		t.Errorf("extra_field not discovered: %+v", h.Fields)
	}
}

func TestBuildUsageHistory(t *testing.T) {
	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	db.SeedDicts(gdb)

	s1 := &model.Snapshot{SnapshotDate: "2026-09-17"}
	s2 := &model.Snapshot{SnapshotDate: "2026-09-18"}
	gdb.Create(s1)
	gdb.Create(s2)
	gdb.Create(&model.TokenSnapshot{SnapshotID: s1.ID, TokenID: 7, TokenName: "claude",
		UsageRaw: mustJSON(t, map[string]interface{}{"total_available": float64(45000000), "total_used": float64(10), "total_granted": float64(100000000)})})
	gdb.Create(&model.TokenSnapshot{SnapshotID: s2.ID, TokenID: 7, TokenName: "claude",
		UsageRaw: mustJSON(t, map[string]interface{}{"total_available": float64(40000000), "total_used": float64(30), "total_granted": float64(100000000)})})

	h := BuildUsageHistory(gdb, 7, 30)
	if len(h.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(h.Rows))
	}
	if h.Rows[1].Values["total_used"] != "30" || h.Rows[1].Deltas["total_used"] != "+20" {
		t.Errorf("bad last row: %+v", h.Rows[1])
	}
	if h.Rows[1].Values["total_available_usd"] != "$80.00" || h.Rows[1].Values["total_granted_usd"] != "$200.00" {
		t.Errorf("missing usage currency values: %+v", h.Rows[1].Values)
	}
	// total_granted 无变化 → +0
	if h.Rows[1].Deltas["total_granted"] != "0" {
		t.Errorf("granted delta = %q, want 0", h.Rows[1].Deltas["total_granted"])
	}
	var foundUSD bool
	for _, f := range h.Fields {
		if f.Path == "total_available_usd" && f.Label == "可用总量（金额）" {
			foundUSD = true
		}
	}
	if !foundUSD {
		t.Errorf("usage currency field missing: %+v", h.Fields)
	}
}

func TestBuildAccountHistoryCurrencyDeltasUsePreviousDaySnapshot(t *testing.T) {
	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SeedDicts(gdb); err != nil {
		t.Fatal(err)
	}
	snaps := []model.Snapshot{
		{
			SnapshotDate: "2026-09-20",
			CreatedAt:    time.Date(2026, 9, 20, 0, 0, 0, 0, time.FixedZone("CST", 8*3600)),
			AccountQuota: 100000000,
			AccountUsed:  50000000,
			AccountRaw:   mustJSON(t, map[string]interface{}{"used_quota": 50000000}),
		},
		{
			SnapshotDate: "2026-09-21",
			CreatedAt:    time.Date(2026, 9, 21, 0, 0, 0, 0, time.FixedZone("CST", 8*3600)),
			AccountQuota: 120000000,
			AccountUsed:  70000000,
			AccountRaw:   mustJSON(t, map[string]interface{}{"used_quota": 70000000}),
		},
		{
			SnapshotDate: "2026-09-21",
			CreatedAt:    time.Date(2026, 9, 21, 11, 6, 0, 0, time.FixedZone("CST", 8*3600)),
			AccountQuota: 130000000,
			AccountUsed:  80000000,
			AccountRaw:   mustJSON(t, map[string]interface{}{"used_quota": 80000000}),
		},
	}
	for _, snap := range snaps {
		if err := gdb.Create(&snap).Error; err != nil {
			t.Fatal(err)
		}
	}

	h := BuildAccountHistory(gdb, 0)
	last := h.Rows[len(h.Rows)-1]
	if last.Values["balance_usd"] != "$260.00" || last.Values["used_usd"] != "$160.00" {
		t.Fatalf("bad currency values: %+v", last.Values)
	}
	if last.Deltas["balance_usd"] != "+$60.00" || last.Deltas["used_usd"] != "+$60.00" {
		t.Fatalf("currency deltas should compare with previous day snapshot: %+v", last.Deltas)
	}
	if last.Deltas["used_quota"] != "+10000000" {
		t.Fatalf("regular fields should still compare adjacent captures: %+v", last.Deltas)
	}
}

func TestSameDayHistoryKeepsCaptureTimesAndDeltas(t *testing.T) {
	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	db.SeedDicts(gdb)
	for i := 0; i < 8; i++ {
		snap := model.Snapshot{SnapshotDate: "2026-09-18", CreatedAt: time.Date(2026, 9, 18, 10, i, 0, 0, time.FixedZone("CST", 8*3600)), AccountRaw: mustJSON(t, map[string]interface{}{"used_quota": 100 + i*10})}
		if err := gdb.Create(&snap).Error; err != nil {
			t.Fatal(err)
		}
		if err := gdb.Create(&model.TokenSnapshot{SnapshotID: snap.ID, TokenID: 7, UsageRaw: mustJSON(t, map[string]interface{}{"total_used": i * 5})}).Error; err != nil {
			t.Fatal(err)
		}
	}
	account := BuildAccountHistory(gdb, 0)
	usage := BuildUsageHistory(gdb, 7, 0)
	for _, h := range []*History{account, usage} {
		if len(h.Rows) != 8 {
			t.Fatalf("rows = %d", len(h.Rows))
		}
		for i, row := range h.Rows {
			if row.CapturedAt.Minute() != i {
				t.Errorf("capture time lost: %v", row.CapturedAt)
			}
			if i > 0 && row.ID <= h.Rows[i-1].ID {
				t.Error("unstable same-day order")
			}
		}
	}
	if account.Rows[7].Deltas["used_quota"] != "+10" || usage.Rows[7].Deltas["total_used"] != "+5" {
		t.Fatal("same-day diff incorrect")
	}
	latest, _ := db.LatestSnapshot(gdb)
	previous, err := db.PreviousSnapshot(gdb, latest)
	if err != nil || previous.ID != account.Rows[6].ID {
		t.Fatal("previous same-day snapshot not selected")
	}
	daily, _ := db.SnapshotByDate(gdb, "2026-09-18")
	if daily.ID != latest.ID {
		t.Fatal("date lookup must pick last capture")
	}
}
