package scheduler

import (
	"testing"
	"time"
)

func TestTimeToCron(t *testing.T) {
	expr, err := timeToCron("10:30")
	if err != nil {
		t.Fatal(err)
	}
	if expr != "0 30 10 * * *" {
		t.Errorf("expr = %s", expr)
	}
}

func TestTimeToCronExport(t *testing.T) {
	expr, err := TimeToCron("00:00")
	if err != nil {
		t.Fatal(err)
	}
	if expr != "0 0 0 * * *" {
		t.Errorf("expr = %s", expr)
	}
}

func TestTimeToCronInvalid(t *testing.T) {
	if _, err := timeToCron("bad"); err == nil {
		t.Error("expected error for invalid time")
	}
}

func TestRegisterAndTrigger(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	s := New(nil, loc)
	done := make(chan bool, 1)
	s.OnSnapshot = func() { done <- true }
	if err := s.RegisterSnapshot("* * * * * *"); err != nil {
		t.Fatal(err)
	}
	s.Start()
	defer s.Stop()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("snapshot task not triggered")
	}
}

func TestRegisterReport(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	s := New(nil, loc)
	done := make(chan bool, 1)
	s.OnReport = func() { done <- true }
	if err := s.RegisterReport("* * * * * *"); err != nil {
		t.Fatal(err)
	}
	s.Start()
	defer s.Stop()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("report task not triggered")
	}
}
