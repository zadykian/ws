package fresh

import (
	"encoding/json"
	"testing"

	"github.com/zadykian/ws/internal/editor"
)

func TestTheThemeIsStrictJSONUnderTheNeutralName(t *testing.T) {
	var theme map[string]any
	if err := json.Unmarshal(Theme, &theme); err != nil {
		t.Fatal(err)
	}
	settings, err := editor.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if theme["name"] != settings.Theme.Dark {
		t.Fatalf("name %v, want %s", theme["name"], settings.Theme.Dark)
	}
	for section, value := range theme {
		colours, isMap := value.(map[string]any)
		if !isMap {
			continue
		}
		for name, colour := range colours {
			triple, isList := colour.([]any)
			if !isList || len(triple) != 3 {
				t.Errorf("%s.%s: %v isn't [r, g, b]", section, name, colour)
			}
		}
	}
}
