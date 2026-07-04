// Package sidecarscript embeds the Python deezload.py sidecar script as a
// string constant. This lets the production binary be self-contained
// (no sidecar/ directory needs to be shipped next to it).
//
// The original editable copy lives at ../../sidecar/deezload.py. When
// changes are made there, copy them into this directory before building.
package sidecarscript

import _ "embed"

//go:embed deezload.py
var Source string
