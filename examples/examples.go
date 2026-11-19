// Package examples holds team manifests that ship with go-dark. The
// belgiantrain manifest doubles as the built-in default team, so a fresh
// profile has the team's projects before any manifest repo is subscribed.
package examples

import _ "embed"

// Belgiantrain is examples/belgiantrain/go-dark.json.
//
//go:embed belgiantrain/go-dark.json
var Belgiantrain []byte
