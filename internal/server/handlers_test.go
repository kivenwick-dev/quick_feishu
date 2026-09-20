package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"quick-feishu/internal/app"
	"quick-feishu/internal/config"
	"quick-feishu/internal/db"
	"quick-feishu/internal/model"
)

func newTestHandlers(t *testing.T) *Handlers {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Default()
	a, err := app.New(cfg, t.TempDir(), filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return &Handlers{App: a}
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
	h.App.DB().Create(&model.Snapshot{SnapshotDate: "2026-09-17", AccountUsed: 100, RequestCount: 5})
	h.App.DB().Create(&model.Snapshot{SnapshotDate: "2026-09-18", AccountUsed: 150, RequestCount: 9})
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
	if err := db.SeedDicts(h.App.DB()); err != nil {
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

func TestSPARootNoRedirectLoop(t *testing.T) {
	h := newTestHandlers(t)
	s := New(0)
	s.RegisterRoutes(h)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	s.Engine.ServeHTTP(w, req)
	if w.Code == http.StatusMovedPermanently || w.Code == http.StatusFound {
		t.Fatalf("root must not redirect (301/302 loop), got %d location=%q", w.Code, w.Header().Get("Location"))
	}
}

func TestUnknownAPIReturns404(t *testing.T) {
	h := newTestHandlers(t)
	s := New(0)
	s.RegisterRoutes(h)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/does-not-exist", nil)
	s.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("unknown api should be 404, got %d", w.Code)
	}
}

func TestGetQuotaRate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/status" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"quota_per_unit":500000,"display_in_currency":true},"success":true}`))
	}))
	defer srv.Close()

	cfg := config.Default()
	cfg.Account.APIBase = srv.URL
	a, err := app.New(cfg, t.TempDir(), filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	h := &Handlers{App: a}
	s := New(0)
	s.RegisterRoutes(h)
	w := httptest.NewRecorder()
	s.Engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/quota-rate", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"quota_per_unit":500000`) {
		t.Fatalf("code = %d body = %s", w.Code, w.Body.String())
	}
}

func TestGetLiveBilling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user/self":
			_, _ = w.Write([]byte(`{"data":{"id":822572,"quota":7526436992,"used_quota":24299227753},"success":true}`))
		case "/api/status":
			_, _ = w.Write([]byte(`{"data":{"quota_per_unit":500000,"display_in_currency":true},"success":true}`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	cfg := config.Default()
	cfg.Account.UserID = "822572"
	cfg.Account.APIBase = srv.URL
	a, err := app.New(cfg, t.TempDir(), filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	h := &Handlers{App: a}
	s := New(0)
	s.RegisterRoutes(h)
	w := httptest.NewRecorder()
	s.Engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/billing/live", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("code = %d body = %s", w.Code, w.Body.String())
	}
	for _, want := range []string{`"quota":7526436992`, `"used_quota":24299227753`, `"quota_per_unit":500000`, `"balance_usd":15052.873984`, `"used_usd":48598.455506`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("missing %s in %s", want, w.Body.String())
		}
	}
}

func TestHandlersFollowAccountDatabaseSwitch(t *testing.T) {
	for _, key := range []string{"QR_USER_ID", "QR_SYSTEM_TOKEN", "QR_FEISHU_WEBHOOK"} {
		t.Setenv(key, "")
	}
	gin.SetMode(gin.TestMode)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	cfg := config.Default()
	cfg.Account.UserID = "acct-a"
	cfg.Account.APIBase = srv.URL
	a, err := app.New(cfg, dir, filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	h := &Handlers{App: a}
	if err := a.DB().Create(&model.Snapshot{SnapshotDate: "2026-09-18", AccountUsed: 777}).Error; err != nil {
		t.Fatal(err)
	}
	s := New(0)
	s.RegisterRoutes(h)

	get := func() string {
		w := httptest.NewRecorder()
		s.Engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/snapshots", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("code = %d body = %s", w.Code, w.Body.String())
		}
		return w.Body.String()
	}

	if body := get(); !strings.Contains(body, `"account_used":777`) {
		t.Fatalf("before switch account A data missing: %s", body)
	}

	next := config.Default()
	next.Account.UserID = "acct-b"
	next.Account.APIBase = srv.URL
	if err := a.SaveConfig(next); err != nil {
		t.Fatal(err)
	}
	if err := a.Restart(); err != nil {
		t.Fatal(err)
	}

	if body := get(); strings.Contains(body, `"account_used":777`) {
		t.Fatalf("after switch account A data still visible: %s", body)
	}
}
