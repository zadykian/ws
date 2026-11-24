package values

import (
	"strings"
	"testing"
)

// The host's roles that the guest runs too read the sections by the guest's names, so the host's
// keys are the guest's by name and kind.
func TestHostKeysAreTheGuestsByName(t *testing.T) {
	guest := map[string]Kind{}
	for _, k := range Keys() {
		guest[k.Name] = k.Kind
	}
	for _, k := range HostKeys() {
		if kind, ok := guest[k.Name]; !ok || kind != k.Kind {
			t.Errorf("host key %s of kind %v: the guest's is %v, %v", k.Name, k.Kind, kind, ok)
		}
	}
}

func TestHostYAMLReadsBackAndSetFindsEachLine(t *testing.T) {
	text, err := DefaultHost().YAML()
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseHost([]byte(text))
	if err != nil || !hostAlike(t, got, DefaultHost()) {
		t.Fatalf("the text reads as %+v, %v; want the defaults:\n%s", got, err, text)
	}
	for _, k := range HostKeys() {
		if value := got.Text(k); value != "" {
			if next, err := SetHost([]byte(text), k.Name, value); err != nil || string(next) != text {
				t.Errorf("set %s %q: %v; want the file unchanged:\n%s", k.Name, value, err, next)
			}
		}
	}
}

func TestSetHostEditsOneLineAndRefusesAGuestsKey(t *testing.T) {
	got, err := SetHost([]byte("---\nWS_BASE_TIMEZONE: UTC\n"), "WS_BASE_SWAPPINESS", "20")
	if want := "---\nWS_BASE_TIMEZONE: UTC\nWS_BASE_SWAPPINESS: 20\n"; err != nil || string(got) != want {
		t.Errorf("SetHost = %q, %v, want %q", got, err, want)
	}
	if _, err := SetHost([]byte("---\n"), "WS_ZED_NOFILE", "1"); err == nil || !strings.Contains(err.Error(), "no line holds WS_ZED_NOFILE") {
		t.Errorf("SetHost of a guest's key = %v, want it refused", err)
	}
}
