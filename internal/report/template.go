package report

import (
	"gopkg.in/yaml.v3"
)

type Template struct {
	Title     string    `yaml:"title" json:"title"`
	DateMode  string    `yaml:"date_mode" json:"date_mode"` // auto | manual
	StartDate string    `yaml:"start_date,omitempty" json:"start_date,omitempty"`
	EndDate   string    `yaml:"end_date,omitempty" json:"end_date,omitempty"`
	Sections  []Section `yaml:"sections" json:"sections"`
}

type Section struct {
	Name     string  `yaml:"section" json:"section"`
	Source   string  `yaml:"source" json:"source"` // account | token | usage
	PerToken bool    `yaml:"per_token" json:"per_token"`
	Fields   []Field `yaml:"fields" json:"fields"`
}

type Field struct {
	Field string `yaml:"field" json:"field"`
	Diff  bool   `yaml:"diff" json:"diff"`
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
		Title:    "AI 平台日报",
		DateMode: "auto",
		Sections: []Section{
			{
				Name:   "账号概况",
				Source: "account",
				Fields: []Field{
					{Field: "used_quota", Diff: true},
					{Field: "request_count", Diff: true},
				},
			},
			{
				Name:     "各令牌用量",
				Source:   "usage",
				PerToken: true,
				Fields: []Field{
					{Field: "total_used", Diff: true},
				},
			},
		},
	}
}
