package backend

import (
	"testing"
)

func TestParseSpotifyURI(t *testing.T) {
	cases := []struct {
		input        string
		expectedType string
		expectedID   string
		shouldError  bool
	}{
		{"https://open.spotify.com/track/4cOdK2wGLETKBW3PvgPWqT", "track", "4cOdK2wGLETKBW3PvgPWqT", false},
		{"spotify:track:4cOdK2wGLETKBW3PvgPWqT", "track", "4cOdK2wGLETKBW3PvgPWqT", false},
		{"https://open.spotify.com/album/1A3nVEWRJ8ywmHZAkewNCD", "album", "1A3nVEWRJ8ywmHZAkewNCD", false},
		{"spotify:playlist:37i9dQZF1DXcBWIGoYBM5M", "playlist", "37i9dQZF1DXcBWIGoYBM5M", false},
		{"https://open.spotify.com/artist/3TVXtAsR1Inumwj472S9r4", "artist", "3TVXtAsR1Inumwj472S9r4", false},
		{"https://open.spotify.com/invalid/123", "", "", true},
		{"invalid_url", "", "", true},
	}

	for _, c := range cases {
		parsed, err := parseSpotifyURI(c.input)
		if c.shouldError {
			if err == nil {
				t.Errorf("parseSpotifyURI(%q) expected error, got nil", c.input)
			}
			continue
		}

		if err != nil {
			t.Errorf("parseSpotifyURI(%q) failed: %v", c.input, err)
			continue
		}

		if parsed.Type != c.expectedType {
			t.Errorf("parseSpotifyURI(%q).Type == %q, expected %q", c.input, parsed.Type, c.expectedType)
		}

		if parsed.ID != c.expectedID {
			t.Errorf("parseSpotifyURI(%q).ID == %q, expected %q", c.input, parsed.ID, c.expectedID)
		}
	}
}
