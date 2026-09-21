package report

import (
	"encoding/json"
	"strings"
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

// DivText 纯文本/富文本区块
type DivText struct {
	Tag  string   `json:"tag"`
	Text CardText `json:"text"`
}

// DivElement 字段区块（保留兼容，当前卡片不再使用并排字段）
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

// Metric 一个指标（三级）；HasDelta 时其下再挂一个「（变动）」子级
type Metric struct {
	Label      string `json:"label"`
	Value      string `json:"value"`
	Delta      string `json:"delta"`
	DeltaLabel string `json:"delta_label"`
	HasDelta   bool   `json:"has_delta"`
}

// TokenNode 一个令牌节点（二级），含其指标（三级）
type TokenNode struct {
	Name    string   `json:"name"`
	Metrics []Metric `json:"metrics"`
}

// SectionResult 一个分区的渲染结果：Fields 为扁平分区，Tokens 为树状分区
type SectionResult struct {
	Name   string       `json:"name"`
	Fields []DiffResult `json:"fields"`
	Tokens []TokenNode  `json:"tokens"`
}

const indent = "\u3000" // 全角空格

// flatContent 扁平分区分内容：一个指标一行；开启差值的指标下方加「（变动）」子级
func flatContent(fields []DiffResult) string {
	var lines []string
	for _, r := range fields {
		if r.Label == "" {
			continue
		}
		lines = append(lines, "**"+r.Label+"**："+r.Value)
		if r.IsDiff && r.Delta != "" {
			lines = append(lines, indent+"└─ "+r.Label+"（"+displayDeltaLabel(r.DeltaLabel)+"）："+r.Delta)
		}
	}
	return strings.Join(lines, "\n")
}

// treeContent 树状分区分内容：分区(一级) → 令牌(二级) → 指标(三级) → 变动(指标子级)
func treeContent(tokens []TokenNode) string {
	var lines []string
	for i, tok := range tokens {
		last := i == len(tokens)-1
		branch := "├─ "
		childPrefix := "│" + indent
		if last {
			branch = "└─ "
			childPrefix = indent + indent
		}
		lines = append(lines, branch+"**"+tok.Name+"**")
		for j, m := range tok.Metrics {
			metricLast := j == len(tok.Metrics)-1
			mbranch := "├─ "
			if metricLast {
				mbranch = "└─ "
			}
			lines = append(lines, childPrefix+mbranch+m.Label+"："+m.Value)
			if m.HasDelta && m.Delta != "" {
				cont := "│" + indent
				if metricLast {
					cont = indent + indent
				}
				lines = append(lines, childPrefix+cont+"└─ "+m.Label+"（"+displayDeltaLabel(m.DeltaLabel)+"）："+m.Delta)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func displayDeltaLabel(label string) string {
	if label == "" {
		return "变动"
	}
	return label
}

func sectionContent(sec SectionResult) string {
	if len(sec.Tokens) > 0 {
		return treeContent(sec.Tokens)
	}
	return flatContent(sec.Fields)
}

func hasContent(sec SectionResult) bool {
	return len(sec.Fields) > 0 || len(sec.Tokens) > 0
}

// BuildCard 根据模板与分区结果生成卡片
func BuildCard(tmpl *Template, date string, sections []SectionResult) (*Card, error) {
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
	first := true
	for _, sec := range sections {
		if !hasContent(sec) {
			continue
		}
		if !first {
			card.Card.Elements = append(card.Card.Elements, HR{Tag: "hr"})
		}
		first = false
		if sec.Name != "" {
			card.Card.Elements = append(card.Card.Elements, DivText{
				Tag:  "div",
				Text: CardText{Tag: "lark_md", Content: "**" + sec.Name + "**"},
			})
		}
		card.Card.Elements = append(card.Card.Elements, DivText{
			Tag:  "div",
			Text: CardText{Tag: "lark_md", Content: sectionContent(sec)},
		})
	}
	return card, nil
}

func (c *Card) ToJSON() ([]byte, error) {
	return json.Marshal(c)
}
