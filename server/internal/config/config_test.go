package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSyllabusItemKeys(t *testing.T) {
	cats := []SyllabusCategoryConfig{
		{Key: "gunplay", Label: "枪法与身法", Items: []SyllabusItemConfig{
			{Key: "pre_aim", Label: "预瞄"},
			{Key: "peek", Label: "闪身枪"},
		}},
		{Key: "awareness", Label: "意识", Items: []SyllabusItemConfig{
			{Key: "sound_sense", Label: "听声辨位"},
		}},
	}
	keys := SyllabusItemKeys(cats)
	if len(keys) != 3 || !keys["pre_aim"] || !keys["peek"] || !keys["sound_sense"] {
		t.Fatalf("unexpected keys: %v", keys)
	}
	if keys["nope"] {
		t.Fatalf("should not contain unknown key")
	}
}

// TestSyllabusJSONCasingIsLowercase 是回归测试：SyllabusCategoryConfig/SyllabusItemConfig
// 序列化为 HTTP 响应 JSON 时必须是小写字段名（key/label/desc/items），
// 与前端 src/api/teaching.ts 里的 SyllabusItem/SyllabusCategory 类型对齐。
// 之前字段没有显式 json 标签，encoding/json 会用大写的 Go 字段名，
// 导致前端拿到的 cat.items 是 undefined，评分表完全渲染不出来。
func TestSyllabusJSONCasingIsLowercase(t *testing.T) {
	cats := []SyllabusCategoryConfig{
		{Key: "gunplay", Label: "枪法与身法", Items: []SyllabusItemConfig{
			{Key: "pre_aim", Label: "预瞄", Desc: "移动时准星保持在头部高度"},
		}},
	}
	b, err := json.Marshal(cats)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	out := string(b)

	for _, want := range []string{`"key"`, `"label"`, `"desc"`, `"items"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected lowercase key %s in JSON output, got: %s", want, out)
		}
	}
	for _, unwanted := range []string{`"Key"`, `"Label"`, `"Desc"`, `"Items"`} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("unexpected capitalized key %s in JSON output, got: %s", unwanted, out)
		}
	}
}
