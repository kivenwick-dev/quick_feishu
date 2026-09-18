package report

import (
	"encoding/json"
)

type Card struct {
	MsgType string   `json:"msg_type"`
	Card    CardBody `json:"card"`
}

type CardBody struct {
	Config   CardConfig    `json:"config"`
	Header   CardHeader    `json:"header"`
	Elements []interface{} `json:"elements"`
}

type CardConfig struct {
	WideScreenMode bool `json:"wide_screen_mode"`
}

type CardHeader struct {
	Title    CardText `json:"title"`
	Template string   `json:"template"`
}

type CardText struct {
	Tag     string `json:"tag"`
	Content string `json:"content"`
}

type DivElement struct {
	Tag    string      `json:"tag"`
	Fields []CardField `json:"fields"`
}

type CardField struct {
	IsShort bool     `json:"is_short"`
	Text    CardText `json:"text"`
}

type HR struct {
	Tag string `json:"tag"`
}

// BuildCard 根据模板与字段结果生成卡片
func BuildCard(tmpl *Template, date string, sections [][]DiffResult) (*Card, error) {
	card := &Card{
		MsgType: "interactive",
		Card: CardBody{
			Config: CardConfig{WideScreenMode: true},
			Header: CardHeader{
				Title:    CardText{Tag: "plain_text", Content: tmpl.Title + " " + date},
				Template: "blue",
			},
			Elements: []interface{}{},
		},
	}
	for i, sec := range sections {
		if len(sec) == 0 {
			continue
		}
		div := DivElement{Tag: "div", Fields: []CardField{}}
		for _, r := range sec {
			content := "**" + r.Label + "**\n" + r.Value
			div.Fields = append(div.Fields, CardField{
				IsShort: true,
				Text:    CardText{Tag: "lark_md", Content: content},
			})
		}
		card.Card.Elements = append(card.Card.Elements, div)
		if i < len(sections)-1 {
			card.Card.Elements = append(card.Card.Elements, HR{Tag: "hr"})
		}
	}
	return card, nil
}

func (c *Card) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}
