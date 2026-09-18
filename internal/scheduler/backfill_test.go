package scheduler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"quick-feishu/internal/api"
	"quick-feishu/internal/collector"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
)

func TestBackfillMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/self":
			w.Write([]byte(`{"data":{"id":1,"quota":1000,"used_quota":100},"success":true}`))
		case "/api/token/":
			w.Write([]byte(`{"data":{"items":[]},"success":true}`))
		case "/api/usage/token/":
			w.Write([]byte(`{"data":{},"success":true}`))
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	gdb, err := db.Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	gdb.Create(&model.Snapshot{SnapshotDate: "2026-09-16", AccountUsed: 10})

	c := api.NewClient(srv.URL, "sys", "1")
	collectFn := func() *collector.Result { return collector.Collect(c) }

	if err := BackfillMissing(gdb, c, "2026-09-18", collectFn); err != nil {
		t.Fatal(err)
	}
	var dates []string
	gdb.Model(&model.Snapshot{}).Order("snapshot_date ASC").Pluck("snapshot_date", &dates)
	if len(dates) != 3 {
		t.Errorf("dates = %v, want 3 (09-16,17,18)", dates)
	}
}

func TestBackfillNoExisting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/self":
			w.Write([]byte(`{"data":{"id":1},"success":true}`))
		case "/api/token/":
			w.Write([]byte(`{"data":{"items":[]},"success":true}`))
		default:
			w.Write([]byte(`{"data":{},"success":true}`))
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	gdb, _ := db.Init(dir)
	c := api.NewClient(srv.URL, "sys", "1")
	collectFn := func() *collector.Result { return collector.Collect(c) }

	if err := BackfillMissing(gdb, c, "2026-09-18", collectFn); err != nil {
		t.Fatal(err)
	}
	var count int64
	gdb.Model(&model.Snapshot{}).Count(&count)
	if count != 1 {
		t.Errorf("count = %d, want 1 (only today)", count)
	}
}

func TestBackfillAlreadyCurrent(t *testing.T) {
	dir := t.TempDir()
	gdb, _ := db.Init(dir)
	gdb.Create(&model.Snapshot{SnapshotDate: "2026-09-18"})
	calls := 0
	collectFn := func() *collector.Result {
		calls++
		return &collector.Result{Usages: map[int]*api.TokenUsageData{}, TokenList: &api.TokenListData{Items: []api.TokenItem{}}}
	}
	if err := BackfillMissing(gdb, nil, "2026-09-18", collectFn); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Errorf("should not collect when already current, calls = %d", calls)
	}
}
