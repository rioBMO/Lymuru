package backend

import (
	"strings"
	"unicode"

	"github.com/gojp/kana"
	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
	"github.com/mozillazg/go-pinyin"
	korean "github.com/srevinsaju/korean-romanizer-go"
)

var (
	jpTokenizer *tokenizer.Tokenizer
)

func init() {
	if t, err := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos()); err == nil {
		jpTokenizer = t
	}
}

// DetectLanguage returns "ja", "ko", "zh", or "" by checking CJK ranges.
func DetectLanguage(text string) string {
	var hasHiragana, hasKatakana, hasHangul, hasHan bool

	for _, r := range text {
		switch {
		case unicode.In(r, unicode.Hiragana):
			hasHiragana = true
		case unicode.In(r, unicode.Katakana):
			hasKatakana = true
		case unicode.In(r, unicode.Hangul):
			hasHangul = true
		case unicode.In(r, unicode.Han):
			hasHan = true
		}
	}

	if hasHangul {
		return "ko"
	}
	if hasHiragana || hasKatakana {
		return "ja"
	}
	if hasHan {
		// Could be Japanese or Chinese, but if it has no kana, default to Chinese.
		// (A more advanced detector could check specific vocab).
		return "zh"
	}
	return ""
}

// RomanizeJapanese tokenizes text with Kagome to get Katakana readings, then transliterates to Romaji.
func RomanizeJapanese(text string) string {
	if jpTokenizer == nil {
		return text
	}
	
	tokens := jpTokenizer.Tokenize(text)
	var sb strings.Builder
	
	for _, t := range tokens {
		if t.Class == tokenizer.DUMMY {
			continue
		}
		
		features := t.Features()
		// Kagome features format (IPA dict):
		// 0: POS, 1: POS1, ... 7: Reading (Katakana), 8: Pronunciation
		if len(features) > 7 && features[7] != "*" {
			reading := features[7]
			romaji := kana.KanaToRomaji(reading)
			sb.WriteString(romaji)
		} else {
			// Fallback: If it's already katakana/hiragana, kana can convert it.
			// Otherwise just append the original string.
			if kana.IsKana(t.Surface) {
				sb.WriteString(kana.KanaToRomaji(t.Surface))
			} else {
				sb.WriteString(t.Surface)
			}
		}
		
		// Add space between tokens if it's not punctuation
		if len(features) > 0 && features[0] != "記号" {
			sb.WriteString(" ")
		}
	}
	
	return strings.TrimSpace(sb.String())
}

// RomanizeChinese converts Hanzi to Pinyin.
func RomanizeChinese(text string) string {
	args := pinyin.NewArgs()
	args.Style = pinyin.Tone
	
	// go-pinyin processes character by character
	result := pinyin.Pinyin(text, args)
	
	var sb strings.Builder
	runes := []rune(text)
	idx := 0
	
	for _, r := range runes {
		if unicode.In(r, unicode.Han) {
			if idx < len(result) && len(result[idx]) > 0 {
				sb.WriteString(result[idx][0])
				sb.WriteString(" ")
			} else {
				sb.WriteRune(r)
			}
			idx++
		} else {
			sb.WriteRune(r)
		}
	}
	
	return strings.TrimSpace(sb.String())
}

// RomanizeKorean converts Hangul to Revised Romanization.
func RomanizeKorean(text string) string {
	r := korean.NewRomanizer(text)
	return r.Romanize()
}

// RomanizeLyrics detects the language line-by-line and romanizes it, preserving LRC timestamps.
func RomanizeLyrics(lyrics string) (string, bool) {
	if lyrics == "" {
		return "", false
	}
	
	lines := strings.Split(lyrics, "\n")
	var sb strings.Builder
	changed := false
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			sb.WriteString("\n")
			continue
		}
		
		timestamp := ""
		content := line
		
		// Extract LRC timestamp if present
		if strings.HasPrefix(line, "[") {
			closeIdx := strings.Index(line, "]")
			if closeIdx > 0 {
				timestamp = line[:closeIdx+1]
				content = strings.TrimSpace(line[closeIdx+1:])
			}
		}
		
		lang := DetectLanguage(content)
		var romanized string
		
		switch lang {
		case "ja":
			romanized = RomanizeJapanese(content)
		case "zh":
			romanized = RomanizeChinese(content)
		case "ko":
			romanized = RomanizeKorean(content)
		default:
			romanized = content
		}
		
		if romanized != content {
			changed = true
		}
		
		if timestamp != "" {
			sb.WriteString(timestamp)
		}
		sb.WriteString(romanized)
		sb.WriteString("\n")
	}
	
	return strings.TrimSpace(sb.String()), changed
}
