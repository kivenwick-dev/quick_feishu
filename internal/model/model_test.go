package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSnapshotJSONTags(t *testing.T) {
	b, _ := json.Marshal(Snapshot{SnapshotDate: "2026-09-18", AccountUsed: 5})
	s := string(b)
	if !strings.Contains(s, `"snapshot_date"`) || !strings.Contains(s, `"account_used"`) {
		t.Errorf("expected snake_case json tags, got %s", s)
	}
}
