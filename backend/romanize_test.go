package backend

import (
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		text     string
		expected string
	}{
		{"Hello world", ""},
		{"こんにちは世界", "ja"},
		{"世界", "zh"},
		{"안녕하세요", "ko"},
		{"[00:12.34] こんにちは", "ja"},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := DetectLanguage(tt.text)
			if got != tt.expected {
				t.Errorf("DetectLanguage(%q) = %q; want %q", tt.text, got, tt.expected)
			}
		})
	}
}

func TestRomanizeKorean(t *testing.T) {
	text := "안녕하세요"
	expected := "annyeonghaseyo"
	got := RomanizeKorean(text)
	if got != expected {
		t.Errorf("RomanizeKorean(%q) = %q; want %q", text, got, expected)
	}
}

func TestRomanizeChinese(t *testing.T) {
	text := "世界"
	// go-pinyin output format with Tone might look like "shì jiè"
	got := RomanizeChinese(text)
	if got == "" || got == text {
		t.Errorf("RomanizeChinese(%q) returned empty or original: %q", text, got)
	}
}

func TestRomanizeJapanese(t *testing.T) {
	text := "こんにちは"
	// kana should convert this to "konnichiha" or similar
	got := RomanizeJapanese(text)
	if got == "" || got == text {
		t.Errorf("RomanizeJapanese(%q) returned empty or original: %q", text, got)
	}
}

func TestRomanizeLyrics(t *testing.T) {
	lyrics := "[00:12.00] こんにちは\n[00:15.00] 안녕하세요"
	got, changed := RomanizeLyrics(lyrics)
	if !changed {
		t.Errorf("RomanizeLyrics expected changed=true")
	}
	if got == "" || got == lyrics {
		t.Errorf("RomanizeLyrics returned original: %q", got)
	}
}
