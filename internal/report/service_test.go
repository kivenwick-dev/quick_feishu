package report

import "testing"

func TestTemplateFromMapEmpty(t *testing.T) {
	tmpl, err := TemplateFromMap(nil)
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Title != "AI 平台日报" {
		t.Errorf("empty map should yield default template, got %q", tmpl.Title)
	}
}

func TestTemplateFromMapCustom(t *testing.T) {
	m := map[string]interface{}{
		"title":     "自定义",
		"date_mode": "manual",
		"sections": []interface{}{
			map[string]interface{}{
				"section": "概况",
				"source":  "account",
				"fields": []interface{}{
					map[string]interface{}{"field": "used_quota", "diff": true},
				},
			},
		},
	}
	tmpl, err := TemplateFromMap(m)
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Title != "自定义" || tmpl.DateMode != "manual" {
		t.Errorf("bad template: %+v", tmpl)
	}
	if len(tmpl.Sections) != 1 || tmpl.Sections[0].Fields[0].Field != "used_quota" {
		t.Errorf("bad sections: %+v", tmpl.Sections)
	}
}
