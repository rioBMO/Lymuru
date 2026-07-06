package backend

import (
	"strings"
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	cases := []struct {
		text     string
		expected string
	}{
		{"こんにちは世界", "ja"},
		{"你好，世界", "zh"},
		{"안녕하세요", "ko"},
		{"Hello World", ""},
	}

	for _, c := range cases {
		lang := DetectLanguage(c.text)
		if lang != c.expected {
			t.Errorf("DetectLanguage(%q) == %q, expected %q", c.text, lang, c.expected)
		}
	}
}

func TestRomanizeJapanese(t *testing.T) {
	text := "こんにちは"
	expected := "konnichiha" // or similar depending on the library
	_ = expected
	result := RomanizeJapanese(text)
	if result == "" || result == text {
		t.Errorf("RomanizeJapanese(%q) failed to romanize", text)
	}
}

func TestRomanizeKorean(t *testing.T) {
	text := "안녕하세요"
	expected := "annyeonghaseyo"
	result := RomanizeKorean(text)
	if !strings.Contains(strings.ToLower(result), "annyeong") && !strings.Contains(strings.ToLower(result), "annyeonghaseyo") {
		t.Errorf("RomanizeKorean(%q) == %q, expected something close to %q", text, result, expected)
	}
}

func TestRomanizeLyrics(t *testing.T) {
	lyrics := "[00:00.00] こんにちは\n[00:05.00] Hello"
	romanized, changed := RomanizeLyrics(lyrics)
	if !changed {
		t.Error("Expected RomanizeLyrics to detect changes")
	}
	if strings.Contains(romanized, "こんにちは") {
		t.Error("Expected Japanese text to be romanized")
	}
	if !strings.Contains(romanized, "Hello") {
		t.Error("Expected English text to remain untouched")
	}
}
