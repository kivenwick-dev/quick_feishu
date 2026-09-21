package report

import (
	"gopkg.in/yaml.v3"
)

type Template struct {
	Title         string    `yaml:"title" json:"title"`
	DateMode      string    `yaml:"date_mode" json:"date_mode"` // auto | manual
	StartDate     string    `yaml:"start_date,omitempty" json:"start_date,omitempty"`
	EndDate       string    `yaml:"end_date,omitempty" json:"end_date,omitempty"`
	AlgorithmNote *string   `yaml:"algorithm_note,omitempty" json:"algorithm_note,omitempty"`
	Sections      []Section `yaml:"sections" json:"sections"`
}

type Section struct {
	Name     string  `yaml:"section" json:"section"`
	Source   string  `yaml:"source" json:"source"` // account | token | usage
	PerToken bool    `yaml:"per_token" json:"per_token"`
	Fields   []Field `yaml:"fields" json:"fields"`
}

type Field struct {
	Field    string `yaml:"field" json:"field"`
	Diff     bool   `yaml:"diff" json:"diff"`
	Currency *bool  `yaml:"currency,omitempty" json:"currency,omitempty"`
}

func boolPtr(v bool) *bool { return &v }

func stringPtr(v string) *string { return &v }

const defaultAlgorithmNote = "主值：日报发送时实时采集值；差值：两个 00:00 快照对比。\n余额/可用类 = 前天快照 - 昨天快照，显示为「消耗」；累计/已用/授予类 = 昨天快照 - 前天快照，显示为「变动」。"

func DefaultAlgorithmNote() string {
	return defaultAlgorithmNote
}

func (t *Template) AlgorithmNoteText() string {
	if t == nil || t.AlgorithmNote == nil {
		return defaultAlgorithmNote
	}
	return *t.AlgorithmNote
}

func currencyEligible(field string) bool {
	return field == "balance_usd" || field == "used_usd" ||
		field == "total_available" || field == "total_used" || field == "total_granted"
}

func (f Field) CurrencyEnabled() bool {
	if !currencyEligible(f.Field) {
		return false
	}
	return f.Currency != nil && *f.Currency
}

func ParseTemplate(data []byte) (*Template, error) {
	var t Template
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	if t.DateMode == "" {
		t.DateMode = "auto"
	}
	return &t, nil
}

func DefaultTemplate() *Template {
	return &Template{
		Title:         "AI 平台日报",
		DateMode:      "auto",
		AlgorithmNote: stringPtr(defaultAlgorithmNote),
		Sections: []Section{
			{
				Name:   "账号概况",
				Source: "account",
				Fields: []Field{
					{Field: "balance_usd", Diff: true, Currency: boolPtr(true)},
					{Field: "used_usd", Diff: true, Currency: boolPtr(true)},
					{Field: "used_quota", Diff: true},
					{Field: "request_count", Diff: true},
				},
			},
			{
				Name:     "各令牌用量",
				Source:   "usage",
				PerToken: true,
				Fields: []Field{
					{Field: "total_used", Diff: true, Currency: boolPtr(true)},
				},
			},
		},
	}
}
