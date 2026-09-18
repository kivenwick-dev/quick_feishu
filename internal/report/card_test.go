package report

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildCard(t *testing.T) {
	tmpl := DefaultTemplate()
	sections := []SectionResult{
		{Name: "账号概况", Fields: []DiffResult{
			{Field: "used_quota", Label: "已用配额", Value: "50", IsDiff: true},
			{Field: "request_count", Label: "请求次数", Value: "10", IsDiff: true},
		}},
		{Name: "各令牌用量", Fields: []DiffResult{
			{Field: "total_used", Label: "累计已用", Value: "888", IsDiff: true},
		}},
	}
	card, err := BuildCard(tmpl, "2026-09-18", sections)
	if err != nil {
		t.Fatal(err)
	}
	if card.MsgType != "interactive" {
		t.Errorf("msg_type = %s", card.MsgType)
	}
	if !strings.Contains(card.Card.Header.Title.Content, "2026-09-18") {
		t.Errorf("header title missing date: %s", card.Card.Header.Title.Content)
	}
	if len(card.Card.Elements) == 0 {
		t.Fatal("no elements")
	}
	if _, ok := card.Card.Elements[0].(HR); ok {
		t.Error("first element must not be HR")
	}
	if _, ok := card.Card.Elements[len(card.Card.Elements)-1].(HR); ok {
		t.Error("last element must not be HR")
	}
	b, err := card.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["msg_type"] != "interactive" {
		t.Error("json msg_type missing")
	}
	s := string(b)
	for _, want := range []string{"账号概况", "各令牌用量", "已用配额", "50"} {
		if !strings.Contains(s, want) {
			t.Errorf("card json missing %q: %s", want, s)
		}
	}
}

func TestBuildCardSkipsEmptySections(t *testing.T) {
	tmpl := DefaultTemplate()
	sections := []SectionResult{
		{Name: "空分区", Fields: nil},
		{Name: "有效", Fields: []DiffResult{{Field: "total_used", Label: "累计已用", Value: "1"}}},
	}
	card, _ := BuildCard(tmpl, "2026-09-18", sections)
	if len(card.Card.Elements) == 0 {
		t.Fatal("no elements")
	}
	if _, ok := card.Card.Elements[0].(HR); ok {
		t.Error("empty first section should not leave a leading HR")
	}
	b, _ := card.ToJSON()
	if strings.Contains(string(b), "空分区") {
		t.Error("empty section title should not be rendered")
	}
	if !strings.Contains(string(b), "有效") {
		t.Error("valid section title should be rendered")
	}
}
