package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDoGetHeaders(t *testing.T) {
	var gotAuth, gotUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUser = r.Header.Get("new-api-user")
		w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "sys-token", "827947")
	_, err := c.doGet("/api/user/self", nil, map[string]string{"new-api-user": c.UserID, "Authorization": "Bearer " + c.SystemToken})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer sys-token" {
		t.Errorf("auth = %s", gotAuth)
	}
	if gotUser != "827947" {
		t.Errorf("new-api-user = %s", gotUser)
	}
}

func TestWithRetrySucceeds(t *testing.T) {
	old := sleepFn
	sleepFn = func(d time.Duration) {}
	defer func() { sleepFn = old }()

	calls := 0
	fn := func() ([]byte, error) {
		calls++
		if calls < 2 {
			return nil, fmt.Errorf("boom")
		}
		return []byte(`ok`), nil
	}
	body, err := WithRetry(3, fn)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %s", body)
	}
	if calls != 2 {
		t.Errorf("calls = %d, want 2", calls)
	}
}

func TestWithRetryGivesUp(t *testing.T) {
	old := sleepFn
	sleepFn = func(d time.Duration) {}
	defer func() { sleepFn = old }()

	calls := 0
	fn := func() ([]byte, error) {
		calls++
		return nil, fmt.Errorf("always fails")
	}
	_, err := WithRetry(3, fn)
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	if calls != 4 {
		t.Errorf("calls = %d, want 4 (1 initial + 3 retries)", calls)
	}
}
