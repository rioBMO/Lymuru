package backend

import (
	"path/filepath"
	"testing"
)

func TestNewestPython(t *testing.T) {
	p := func(dir string) string { return filepath.Join("C:", "Python", dir, "python.exe") }

	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{"empty", nil, ""},
		{"single", []string{p("Python313")}, p("Python313")},
		{"numeric not lexical", []string{p("Python39"), p("Python310")}, p("Python310")},
		{"case-insensitive", []string{p("python311"), p("PYTHON312")}, p("PYTHON312")},
		{"suffixes allowed", []string{p("Python312"), p("Python313-32")}, p("Python313-32")},
		{"unparsable ignored", []string{p("Python3"), p("PythonX"), p("Python311")}, p("Python311")},
		{"python2 ignored", []string{p("Python27")}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newestPython(tt.paths); got != tt.want {
				t.Errorf("newestPython() = %q, want %q", got, tt.want)
			}
		})
	}
}
