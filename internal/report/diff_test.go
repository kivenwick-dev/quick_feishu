package report

import (
	"encoding/json"
	"testing"

	"quick-feishu/internal/model"
)

func mkSnap(used int64) *model.Snapshot {
	raw, _ := json.Marshal(map[string]interface{}{"used_quota": used})
	return &model.Snapshot{AccountRaw: raw}
}

func TestComputeDiffNumeric(t *testing.T) {
	early := mkSnap(100)
	late := mkSnap(150)
	res, err := ComputeDiffAccount(early, late, Field{Field: "used_quota", Diff: true}, "已用配额")
	if err != nil {
		t.Fatal(err)
	}
	// 主值始终为今日值，变动为带符号差值
	if res.Value != "150" {
		t.Errorf("value = %s, want 150 (today)", res.Value)
	}
	if !res.IsDiff || res.Delta != "+50" {
		t.Errorf("delta = %q (diff=%v), want +50", res.Delta, res.IsDiff)
	}
}

func TestComputeDiffNegative(t *testing.T) {
	early := mkSnap(150)
	late := mkSnap(100)
	res, _ := ComputeDiffAccount(early, late, Field{Field: "used_quota", Diff: true}, "已用配额")
	if res.Value != "100" || res.Delta != "-50" {
		t.Errorf("value=%q delta=%q, want 100 / -50", res.Value, res.Delta)
	}
}

func TestComputeDiffNoEarly(t *testing.T) {
	late := mkSnap(150)
	res, _ := ComputeDiffAccount(nil, late, Field{Field: "used_quota", Diff: true}, "已用配额")
	if res.IsDiff {
		t.Errorf("no early should not diff")
	}
	if res.Value != "150" {
		t.Errorf("value = %s", res.Value)
	}
	if res.Delta != "" {
		t.Errorf("delta should be empty, got %q", res.Delta)
	}
}

func TestComputeDiffNonNumeric(t *testing.T) {
	rawA, _ := json.Marshal(map[string]interface{}{"group": "A"})
	rawB, _ := json.Marshal(map[string]interface{}{"group": "B"})
	early := &model.Snapshot{AccountRaw: rawA}
	late := &model.Snapshot{AccountRaw: rawB}
	res, _ := ComputeDiffAccount(early, late, Field{Field: "group", Diff: true}, "分组")
	if res.IsDiff {
		t.Errorf("string field should not diff")
	}
	if res.Value != "B" {
		t.Errorf("value = %s", res.Value)
	}
}

func TestComputeDiffNoDiffFlag(t *testing.T) {
	early := mkSnap(100)
	late := mkSnap(150)
	res, _ := ComputeDiffAccount(early, late, Field{Field: "used_quota", Diff: false}, "已用配额")
	if res.IsDiff {
		t.Error("diff=false should not compute diff")
	}
	if res.Value != "150" {
		t.Errorf("value = %s", res.Value)
	}
}

func TestExtractNestedField(t *testing.T) {
	raw, _ := json.Marshal(map[string]interface{}{"a": map[string]interface{}{"b": float64(42)}})
	v, ok := extractField(raw, "a.b")
	if !ok || formatVal(v) != "42" {
		t.Errorf("nested extract failed: %v %v", v, ok)
	}
}
