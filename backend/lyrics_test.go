package backend

import (
	"testing"
)

func TestSearchLRCLIB(t *testing.T) {
	lyrics, synced, err := SearchLRCLIB("Rick Astley", "Never Gonna Give You Up")
	if err != nil {
		t.Fatalf("SearchLRCLIB failed: %v", err)
	}

	if lyrics == "" {
		t.Error("Expected lyrics, got empty string")
	}

	if !synced {
		t.Log("Note: expected synced lyrics for Rick Astley, but got unsynced (this is fine depending on LRCLIB's database)")
	}
}

func TestSimplifyTrackName(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"Song Title (feat. Artist)", "Song Title"},
		{"Song Title - Remastered 2020", "Song Title"},
		{"Song Title (Radio Edit)", "Song Title"},
		{"Normal Song", "Normal Song"},
	}

	for _, c := range cases {
		output := simplifyTrackName(c.input)
		if output != c.expected {
			t.Errorf("simplifyTrackName(%q) == %q, expected %q", c.input, output, c.expected)
		}
	}
}
