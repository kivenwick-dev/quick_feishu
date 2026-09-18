package api

import (
	"errors"
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

func TestWithRetryNoRetryOn4xx(t *testing.T) {
	old := sleepFn
	sleepFn = func(d time.Duration) {}
	defer func() { sleepFn = old }()

	calls := 0
	fn := func() ([]byte, error) {
		calls++
		return nil, &HTTPError{Status: http.StatusUnauthorized, Body: "invalid token"}
	}
	_, err := WithRetry(3, fn)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("4xx must not retry, calls = %d, want 1", calls)
	}

	// 429 同样不重试（避免触发防滥用限流）
	calls = 0
	fn429 := func() ([]byte, error) {
		calls++
		return nil, &HTTPError{Status: http.StatusTooManyRequests, Body: "slow down"}
	}
	if _, err := WithRetry(3, fn429); err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("429 must not retry, calls = %d, want 1", calls)
	}
}

func TestWithRetryRetries5xx(t *testing.T) {
	old := sleepFn
	sleepFn = func(d time.Duration) {}
	defer func() { sleepFn = old }()

	calls := 0
	fn := func() ([]byte, error) {
		calls++
		if calls < 3 {
			return nil, &HTTPError{Status: http.StatusBadGateway, Body: "bad gateway"}
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
	if calls != 3 {
		t.Errorf("5xx should retry, calls = %d, want 3", calls)
	}
}

func TestDoGetReturnsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid token"}}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "sys", "1")
	_, err := c.doGet("/api/usage/token/", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var he *HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("expected *HTTPError, got %T", err)
	}
	if he.Status != http.StatusUnauthorized || he.Retryable() {
		t.Errorf("bad HTTPError: %+v retryable=%v", he, he.Retryable())
	}
}
