package state

import (
	_ "embed"
	"encoding/json"
)

//go:embed default_state.json
var defaultStateBytes []byte

// DefaultStateBytes returns a fresh copy of the embedded initial school state
// so the Go server works out-of-the-box on a clean database or after reset.
func DefaultStateBytes() []byte {
	out := make([]byte, len(defaultStateBytes))
	copy(out, defaultStateBytes)
	return out
}

// DefaultStateMap decodes the embedded default state into a mutable map.
func DefaultStateMap() (map[string]any, error) {
	var doc map[string]any
	if err := json.Unmarshal(defaultStateBytes, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}
