package fresh

import (
	"testing"

	"github.com/zadykian/ws/internal/editor"
)

func TestConfigNamesTheTheme(t *testing.T) {
	settings, err := editor.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if got := typed(t).Theme; got != settings.Theme.Dark {
		t.Errorf("theme %q, want %q", got, settings.Theme.Dark)
	}
}
