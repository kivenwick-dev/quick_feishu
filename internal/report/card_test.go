package report

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildCard(t *testing.T) {
	tmpl := DefaultTemplate()
	sections := [][]DiffResult{
		{
			{Field: "used_quota", Label: "已用配额", Value: "50", IsDiff: true},
			{Field: "request_count", Label: "请求次数", Value: "10", IsDiff: true},
		},
		{
			{Field: "total_used", Label: "累计已用", Value: "888", IsDiff: true},
		},
	}
	card, err := BuildCard(tmpl, "2026-09-18", sections)
	if err != nil {
		t.Fatal(err)
	}
	if card.MsgType != "interactive" {
		t.Errorf("msg_type = %s", card.MsgType)
	}
	if len(card.Card.Elements) < 2 {
		t.Errorf("elements = %d, want >= 2", len(card.Card.Elements))
	}
	if !strings.Contains(card.Card.Header.Title.Content, "2026-09-18") {
		t.Errorf("header title missing date: %s", card.Card.Header.Title.Content)
	}
	// 校验可序列化
	b, err := card.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["msg_type"] != "interactive" {
		t.Errorf("json msg_type missing")
	}
	if !strings.Contains(string(b), "已用配额") || !strings.Contains(string(b), "50") {
		t.Errorf("card json missing field label/value: %s", string(b))
	}
}

func TestBuildCardSkipsEmptySections(t *testing.T) {
	tmpl := DefaultTemplate()
	sections := [][]DiffResult{
		nil,
		{{Field: "total_used", Label: "累计已用", Value: "1"}},
	}
	card, _ := BuildCard(tmpl, "2026-09-18", sections)
	for _, el := range card.Card.Elements {
		if hr, ok := el.(HR); ok {
			t.Errorf("empty first section should not leave a leading HR: %+v", hr)
		}
	}
	if len(card.Card.Elements) != 1 {
		t.Errorf("elements = %d, want 1 (empty section skipped)", len(card.Card.Elements))
	}
}
