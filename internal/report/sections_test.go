package report

import (
	"encoding/json"
	"strings"
	"testing"

	"gorm.io/datatypes"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
)

func mustJSON(t *testing.T, m map[string]interface{}) datatypes.JSON {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return datatypes.JSON(b)
}

func TestBuildSectionsAccountUsesDictLabel(t *testing.T) {
	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SeedDicts(gdb); err != nil {
		t.Fatal(err)
	}

	latest := &model.Snapshot{
		SnapshotDate: "2026-09-18",
		AccountRaw:   mustJSON(t, map[string]interface{}{"username": "wick", "used_quota": float64(100)}),
	}
	gdb.Create(latest)

	tmpl := &Template{Sections: []Section{
		{Name: "主账号信息", Source: "account", Fields: []Field{
			{Field: "username", Diff: false},
			{Field: "used_quota", Diff: false},
			{Field: "not_exist", Diff: false},
		}},
	}}
	secs := BuildSections(gdb, latest, nil, tmpl)
	if len(secs) != 1 {
		t.Fatalf("sections = %d, want 1", len(secs))
	}
	f := secs[0].Fields
	if len(f) != 2 {
		t.Fatalf("fields = %d, want 2 (not_exist skipped): %+v", len(f), f)
	}
	if f[0].Label != "用户名" || f[0].Value != "wick" {
		t.Errorf("bad field 0: %+v", f[0])
	}
	if f[1].Label != "已用配额" || f[1].Value != "100" {
		t.Errorf("bad field 1: %+v", f[1])
	}
}

func TestBuildSectionsAccountDerivedUSDFields(t *testing.T) {
	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SeedDicts(gdb); err != nil {
		t.Fatal(err)
	}

	prev := &model.Snapshot{
		SnapshotDate: "2026-09-17",
		AccountRaw:   mustJSON(t, map[string]interface{}{}),
		AccountQuota: 33500000000,
		AccountUsed:  29300000000,
	}
	latest := &model.Snapshot{
		SnapshotDate: "2026-09-18",
		AccountRaw:   mustJSON(t, map[string]interface{}{}),
		AccountQuota: 33454114092,
		AccountUsed:  29360568243,
	}
	gdb.Create(prev)
	gdb.Create(latest)

	tmpl := &Template{Sections: []Section{
		{Name: "美元账务", Source: "account", Fields: []Field{
			{Field: "balance_usd", Diff: true, Currency: boolPtr(true)},
			{Field: "used_usd", Diff: true, Currency: boolPtr(true)},
		}},
	}}
	secs := BuildSections(gdb, latest, prev, tmpl)
	if len(secs) != 1 {
		t.Fatalf("sections = %d, want 1", len(secs))
	}
	fields := secs[0].Fields
	if len(fields) != 2 {
		t.Fatalf("fields = %d, want 2: %+v", len(fields), fields)
	}
	if fields[0].Label != "当前余额" || fields[0].Value != "$66,908.23" || fields[0].Delta != "+$91.77" || fields[0].DeltaLabel != "消耗" {
		t.Errorf("bad balance field: %+v", fields[0])
	}
	if fields[1].Label != "历史消耗" || fields[1].Value != "$58,721.14" || fields[1].Delta != "+$121.14" || fields[1].DeltaLabel != "变动" {
		t.Errorf("bad used field: %+v", fields[1])
	}

	off := false
	tmpl.Sections[0].Fields = []Field{{Field: "balance_usd", Diff: true, Currency: &off}}
	secs = BuildSections(gdb, latest, prev, tmpl)
	got := secs[0].Fields[0]
	if got.Value != "33454114092" || got.Delta != "+45885908" || got.DeltaLabel != "消耗" {
		t.Errorf("currency disabled should render quota, got %+v", got)
	}
}

func TestBuildSectionsUsageTree(t *testing.T) {
	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SeedDicts(gdb); err != nil {
		t.Fatal(err)
	}

	latest := &model.Snapshot{SnapshotDate: "2026-09-18", AccountRaw: mustJSON(t, map[string]interface{}{})}
	gdb.Create(latest)

	gdb.Create(&model.TokenSnapshot{
		SnapshotID: latest.ID, TokenID: 1, TokenName: "claude", TotalUsed: 5,
		UsageRaw: mustJSON(t, map[string]interface{}{"name": "claude", "total_used": float64(5), "total_available": float64(75), "total_granted": float64(100)}),
	})
	gdb.Create(&model.TokenSnapshot{
		SnapshotID: latest.ID, TokenID: 2, TokenName: "gpt", TotalUsed: 7,
		UsageRaw: mustJSON(t, map[string]interface{}{"name": "gpt", "total_used": float64(7), "total_available": float64(93), "total_granted": float64(100)}),
	})

	tmpl := &Template{Sections: []Section{
		{Name: "key使用情况", Source: "usage", PerToken: true, Fields: []Field{
			{Field: "name", Diff: false},
			{Field: "total_used", Diff: false},
			{Field: "total_available", Diff: false},
		}},
	}}
	secs := BuildSections(gdb, latest, nil, tmpl)
	if len(secs) != 1 {
		t.Fatalf("sections = %d, want 1", len(secs))
	}
	toks := secs[0].Tokens
	if len(toks) != 2 {
		t.Fatalf("tokens = %d, want 2", len(toks))
	}
	if toks[0].Name != "claude" || toks[1].Name != "gpt" {
		t.Errorf("bad node names: %q %q", toks[0].Name, toks[1].Name)
	}
	if len(toks[0].Metrics) != 3 {
		t.Fatalf("metrics = %d, want 3 (name excluded + remaining percent): %+v", len(toks[0].Metrics), toks[0].Metrics)
	}
	if toks[0].Metrics[0].Label != "累计已用" || toks[0].Metrics[0].Value != "5" {
		t.Errorf("bad metric 0: %+v", toks[0].Metrics[0])
	}
	if toks[0].Metrics[1].Label != "可用总量" || toks[0].Metrics[1].Value != "75" {
		t.Errorf("bad metric 1: %+v", toks[0].Metrics[1])
	}
	if toks[0].Metrics[2].Label != "剩余用量" || toks[0].Metrics[2].Value != "75%" {
		t.Errorf("bad remaining percent metric: %+v", toks[0].Metrics[2])
	}
}

func TestBuildSectionsUsageDiff(t *testing.T) {
	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	db.SeedDicts(gdb)

	prev := &model.Snapshot{SnapshotDate: "2026-09-17", AccountRaw: mustJSON(t, map[string]interface{}{})}
	latest := &model.Snapshot{SnapshotDate: "2026-09-18", AccountRaw: mustJSON(t, map[string]interface{}{})}
	gdb.Create(prev)
	gdb.Create(latest)
	gdb.Create(&model.TokenSnapshot{SnapshotID: prev.ID, TokenID: 1, TokenName: "claude", TotalUsed: 10,
		UsageRaw: mustJSON(t, map[string]interface{}{"total_used": float64(10)})})
	gdb.Create(&model.TokenSnapshot{SnapshotID: latest.ID, TokenID: 1, TokenName: "claude", TotalUsed: 30,
		UsageRaw: mustJSON(t, map[string]interface{}{"total_used": float64(30)})})

	tmpl := &Template{Sections: []Section{
		{Name: "用量", Source: "usage", PerToken: true, Fields: []Field{{Field: "total_used", Diff: true}}},
	}}
	secs := BuildSections(gdb, latest, prev, tmpl)
	if len(secs) != 1 || len(secs[0].Tokens) != 1 {
		t.Fatalf("unexpected sections: %+v", secs)
	}
	m := secs[0].Tokens[0].Metrics[0]
	if m.Label != "累计已用" || m.Value != "30" || !m.HasDelta || m.Delta != "+20" {
		t.Errorf("expected today=30 delta=+20, got %+v", m)
	}
}

func TestBuildSectionsUsageCurrencySwitch(t *testing.T) {
	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	db.SeedDicts(gdb)

	prev := &model.Snapshot{SnapshotDate: "2026-09-17", AccountRaw: mustJSON(t, map[string]interface{}{})}
	latest := &model.Snapshot{SnapshotDate: "2026-09-18", AccountRaw: mustJSON(t, map[string]interface{}{})}
	gdb.Create(prev)
	gdb.Create(latest)
	gdb.Create(&model.TokenSnapshot{SnapshotID: prev.ID, TokenID: 1, TokenName: "claude",
		UsageRaw: mustJSON(t, map[string]interface{}{"total_used": float64(500000), "total_available": float64(1500000)})})
	gdb.Create(&model.TokenSnapshot{SnapshotID: latest.ID, TokenID: 1, TokenName: "claude",
		UsageRaw: mustJSON(t, map[string]interface{}{"total_used": float64(1000000), "total_available": float64(1000000)})})

	on := true
	off := false
	tmpl := &Template{Sections: []Section{
		{Name: "用量", Source: "usage", PerToken: true, Fields: []Field{{Field: "total_used", Diff: true, Currency: &on}}},
	}}
	overrides := map[int]map[string]interface{}{1: map[string]interface{}{"total_used": int64(1500000)}}
	secs := BuildSectionsWithOverrides(gdb, latest, prev, tmpl, nil, overrides, 500000)
	got := secs[0].Tokens[0].Metrics[0]
	if got.Value != "$3.00" || got.Delta != "+$1.00" {
		t.Errorf("currency enabled bad metric: %+v", got)
	}

	tmpl.Sections[0].Fields[0].Currency = &off
	secs = BuildSectionsWithOverrides(gdb, latest, prev, tmpl, nil, overrides, 500000)
	got = secs[0].Tokens[0].Metrics[0]
	if got.Value != "1500000" || got.Delta != "+500000" {
		t.Errorf("currency disabled bad metric: %+v", got)
	}

	tmpl.Sections[0].Fields = []Field{{Field: "total_available", Diff: true, Currency: &on}}
	overrides = map[int]map[string]interface{}{1: map[string]interface{}{"total_available": int64(2000000)}}
	secs = BuildSectionsWithOverrides(gdb, latest, prev, tmpl, nil, overrides, 500000)
	got = secs[0].Tokens[0].Metrics[0]
	if got.Value != "$4.00" || got.Delta != "+$1.00" || got.DeltaLabel != "消耗" {
		t.Errorf("available currency metric should show consumption, got %+v", got)
	}
}

func TestBuildSectionsRemainingPercentUsesLiveOverrides(t *testing.T) {
	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	db.SeedDicts(gdb)

	latest := &model.Snapshot{SnapshotDate: "2026-09-18", AccountRaw: mustJSON(t, map[string]interface{}{})}
	gdb.Create(latest)
	gdb.Create(&model.TokenSnapshot{SnapshotID: latest.ID, TokenID: 1, TokenName: "claude",
		UsageRaw: mustJSON(t, map[string]interface{}{"total_available": float64(50), "total_granted": float64(100)})})

	tmpl := &Template{Sections: []Section{
		{Name: "用量", Source: "usage", PerToken: true, Fields: []Field{{Field: remainingPercentField}}},
	}}
	overrides := map[int]map[string]interface{}{1: map[string]interface{}{"total_available": int64(300), "total_granted": int64(450)}}
	secs := BuildSectionsWithOverrides(gdb, latest, nil, tmpl, nil, overrides, 500000)
	got := secs[0].Tokens[0].Metrics[0]
	if got.Label != "剩余用量" || got.Value != "67%" {
		t.Errorf("remaining percent should use live overrides and round to integer, got %+v", got)
	}
	if len(secs[0].Tokens[0].Metrics) != 1 {
		t.Errorf("remaining percent should not be duplicated when selected in template: %+v", secs[0].Tokens[0].Metrics)
	}
}

func TestTreeContent(t *testing.T) {
	toks := []TokenNode{
		{Name: "claude", Metrics: []Metric{
			{Label: "累计已用", Value: "5", Delta: "+2", HasDelta: true},
			{Label: "可用总量", Value: "95"},
		}},
		{Name: "gpt", Metrics: []Metric{{Label: "累计已用", Value: "7"}}},
	}
	s := treeContent(toks)
	for _, want := range []string{
		"├─ **claude**",
		"└─ **gpt**",
		"累计已用：5",
		"└─ 累计已用（变动）：+2",
		"└─ 累计已用：7",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestFlatContentWithDelta(t *testing.T) {
	s := flatContent([]DiffResult{
		{Label: "总配额", Value: "6459941149", Delta: "+123", DeltaLabel: "消耗", IsDiff: true},
		{Label: "请求次数", Value: "10"},
	})
	for _, want := range []string{"**总配额**：6459941149", "└─ 总配额（消耗）：+123", "**请求次数**：10"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestFlatContentSkipsEmptyLabel(t *testing.T) {
	s := flatContent([]DiffResult{
		{Label: "用户名", Value: "wick"},
		{Label: "", Value: "-"},
	})
	if strings.Contains(s, "****") {
		t.Errorf("empty label should be skipped, got:\n%s", s)
	}
	if !strings.Contains(s, "**用户名**：wick") {
		t.Errorf("missing field line:\n%s", s)
	}
}
