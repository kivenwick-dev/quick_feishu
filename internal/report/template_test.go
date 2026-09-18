package report

import "testing"

func TestParseTemplate(t *testing.T) {
	data := []byte(`
title: 测试日报
date_mode: manual
start_date: "2026-09-17"
end_date: "2026-09-18"
sections:
  - section: 概况
    source: account
    fields:
      - field: used_quota
        diff: true
`)
	tmpl, err := ParseTemplate(data)
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.DateMode != "manual" || tmpl.StartDate != "2026-09-17" {
		t.Errorf("bad template: %+v", tmpl)
	}
	if len(tmpl.Sections) != 1 || tmpl.Sections[0].Fields[0].Field != "used_quota" {
		t.Errorf("bad sections: %+v", tmpl.Sections)
	}
	if !tmpl.Sections[0].Fields[0].Diff {
		t.Error("diff should be true")
	}
}

func TestParseTemplateDefaultsDateMode(t *testing.T) {
	tmpl, err := ParseTemplate([]byte("title: x\n"))
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.DateMode != "auto" {
		t.Errorf("empty date_mode should default to auto, got %s", tmpl.DateMode)
	}
}

func TestDefaultTemplate(t *testing.T) {
	tmpl := DefaultTemplate()
	if tmpl.DateMode != "auto" {
		t.Errorf("default should be auto")
	}
	if len(tmpl.Sections) != 2 {
		t.Errorf("default sections = %d, want 2", len(tmpl.Sections))
	}
	if tmpl.Sections[1].Source != "usage" || !tmpl.Sections[1].PerToken {
		t.Errorf("second section should be per-token usage: %+v", tmpl.Sections[1])
	}
}
