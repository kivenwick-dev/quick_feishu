package feishu

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func stubSleep(t *testing.T) {
	t.Helper()
	sleepFn = func(time.Duration) {}
	t.Cleanup(func() { sleepFn = time.Sleep })
}

func TestSendCardSuccess(t *testing.T) {
	stubSleep(t)
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Write([]byte(`{"code":0,"msg":"success"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2)
	resp, err := c.SendCard([]byte(`{"msg_type":"interactive"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp != `{"code":0,"msg":"success"}` {
		t.Errorf("resp = %s", resp)
	}
	if gotBody != `{"msg_type":"interactive"}` {
		t.Errorf("body = %s", gotBody)
	}
}

func TestSendCardRetryThenFail(t *testing.T) {
	stubSleep(t)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`bad`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3)
	_, err := c.SendCard([]byte(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 4 {
		t.Errorf("calls = %d, want 4 (1 + 3 retries)", calls)
	}
}

func TestSendCardBusinessCode(t *testing.T) {
	stubSleep(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":19001,"msg":"invalid webhook"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 1)
	_, err := c.SendCard([]byte(`{}`))
	if err == nil {
		t.Fatal("business error should be returned")
	}
}

func TestSendCardRetryThenSuccess(t *testing.T) {
	stubSleep(t)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`err`))
			return
		}
		w.Write([]byte(`{"code":0,"msg":"ok"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 3)
	resp, err := c.SendCard([]byte(`{}`))
	if err != nil {
		t.Fatalf("should succeed on 3rd attempt: %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
	if resp != `{"code":0,"msg":"ok"}` {
		t.Errorf("resp = %s", resp)
	}
}
