package config

import "testing"

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
