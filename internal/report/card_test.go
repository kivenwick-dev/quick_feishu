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
	for _, want := range []string{"算法说明", "账号概况", "各令牌用量", "已用配额", "50"} {
		if !strings.Contains(s, want) {
			t.Errorf("card json missing %q: %s", want, s)
		}
	}
}

func TestBuildCardUsesCustomAlgorithmNote(t *testing.T) {
	note := "自定义说明"
	tmpl := &Template{Title: "日报", AlgorithmNote: &note}
	card, err := BuildCard(tmpl, "2026-09-18", []SectionResult{
		{Name: "有效", Fields: []DiffResult{{Field: "total_used", Label: "累计已用", Value: "1"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := card.ToJSON()
	if !strings.Contains(string(b), "自定义说明") {
		t.Fatalf("missing custom algorithm note: %s", string(b))
	}

	blank := ""
	tmpl.AlgorithmNote = &blank
	card, err = BuildCard(tmpl, "2026-09-18", []SectionResult{
		{Name: "有效", Fields: []DiffResult{{Field: "total_used", Label: "累计已用", Value: "1"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	b, _ = card.ToJSON()
	if strings.Contains(string(b), "算法说明") {
		t.Fatalf("blank algorithm note should hide explanation: %s", string(b))
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

func TestBuildCardAddsDefaultTokenSectionTitle(t *testing.T) {
	tmpl := &Template{Title: "日报", AlgorithmNote: stringPtr("")}
	card, err := BuildCard(tmpl, "2026-09-20", []SectionResult{{
		Tokens: []TokenNode{{Name: "claude", Metrics: []Metric{{Label: "累计已用", Value: "$1.00"}}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := card.ToJSON()
	text := string(b)
	if !strings.Contains(text, "**令牌使用情况**") {
		t.Fatalf("missing default token section title: %s", text)
	}
	if !strings.Contains(text, "claude") || !strings.Contains(text, "累计已用") {
		t.Fatalf("token content missing: %s", text)
	}
}

func TestBuildCardSplitsLongTokenContent(t *testing.T) {
	tmpl := &Template{Title: "日报", AlgorithmNote: stringPtr("")}
	tokens := make([]TokenNode, 0, 80)
	for i := 0; i < 80; i++ {
		tokens = append(tokens, TokenNode{
			Name: strings.Repeat("token-", 8) + string(rune('A'+i%26)),
			Metrics: []Metric{
				{Label: "可用总量", Value: "$10.00", Delta: "+$1.00", DeltaLabel: "消耗", HasDelta: true},
				{Label: "累计已用", Value: "$5.00", Delta: "+$1.00", HasDelta: true},
			},
		})
	}
	card, err := BuildCard(tmpl, "2026-09-20", []SectionResult{{Name: "key使用情况", Tokens: tokens}})
	if err != nil {
		t.Fatal(err)
	}
	contentBlocks := 0
	for _, el := range card.Card.Elements {
		div, ok := el.(DivText)
		if !ok || div.Text.Content == "**key使用情况**" {
			continue
		}
		contentBlocks++
		if len(div.Text.Content) > maxLarkMDContentLen {
			t.Fatalf("content block length = %d, want <= %d", len(div.Text.Content), maxLarkMDContentLen)
		}
	}
	if contentBlocks < 2 {
		t.Fatalf("long token content was not split, contentBlocks = %d", contentBlocks)
	}
}
