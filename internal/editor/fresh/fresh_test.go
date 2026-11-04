package fresh

import (
	"bytes"
	"encoding/json"
	"testing"
)

// rendered parses config.json into a map, as a struct's zero value hides a missing key.
func rendered(t *testing.T) map[string]any {
	data, err := Config()
	if err != nil {
		t.Fatal(err)
	}
	body, found := bytes.CutPrefix(data, []byte(header))
	if !found {
		t.Fatal("no header")
	}
	var c map[string]any
	if err := json.Unmarshal(body, &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestConfigTurnsUpdatesAndSessionsOff(t *testing.T) {
	c := rendered(t)
	for _, key := range []string{"check_for_updates", "self_update", "auto_attach"} {
		if got, found := c[key]; !found || got != false {
			t.Errorf("%s = %v, found %v; want false", key, got, found)
		}
	}
}
