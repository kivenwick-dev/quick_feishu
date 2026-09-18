package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
)

func newTestHandlers(t *testing.T) *Handlers {
	t.Helper()
	gin.SetMode(gin.TestMode)
	gdb, err := db.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return &Handlers{DB: gdb}
}

func TestDashboardEmpty(t *testing.T) {
	h := newTestHandlers(t)
	s := New(0)
	s.RegisterRoutes(h)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	s.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("code = %d", w.Code)
	}
}

func TestListSnapshotsEmpty(t *testing.T) {
	h := newTestHandlers(t)
	s := New(0)
	s.RegisterRoutes(h)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/snapshots", nil)
	s.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("code = %d", w.Code)
	}
}

func TestCompareSnapshotsMissing(t *testing.T) {
	h := newTestHandlers(t)
	s := New(0)
	s.RegisterRoutes(h)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/compare?from=2026-09-17&to=2026-09-18", nil)
	s.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing snapshots should be 400, got %d", w.Code)
	}
}

func TestCompareSnapshotsOK(t *testing.T) {
	h := newTestHandlers(t)
	h.DB.Create(&model.Snapshot{SnapshotDate: "2026-09-17", AccountUsed: 100, RequestCount: 5})
	h.DB.Create(&model.Snapshot{SnapshotDate: "2026-09-18", AccountUsed: 150, RequestCount: 9})
	s := New(0)
	s.RegisterRoutes(h)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/compare?from=2026-09-17&to=2026-09-18", nil)
	s.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"used":50`) {
		t.Errorf("expected diff used 50, body=%s", body)
	}
}

func TestGetDictAccount(t *testing.T) {
	h := newTestHandlers(t)
	if err := db.SeedDicts(h.DB); err != nil {
		t.Fatal(err)
	}
	s := New(0)
	s.RegisterRoutes(h)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/dict/account", nil)
	s.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("code = %d", w.Code)
	}
}
