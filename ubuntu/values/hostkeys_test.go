package values

import (
	"strings"
	"testing"
)

// The host's roles that the guest runs too read the sections by the guest's names, so a key that
// both hold has the same kind.
func TestHostKeysAreTheGuestsByName(t *testing.T) {
	guest := map[string]Kind{}
	for _, k := range Keys() {
		guest[k.Name] = k.Kind
	}
	for _, k := range HostKeys() {
		if kind, ok := guest[k.Name]; ok && kind != k.Kind {
			t.Errorf("host key %s of kind %v: the guest's is %v", k.Name, k.Kind, kind)
		}
	}
}

// The host's own keys are those of the roles that run on the host alone, fans and cable.
func TestHostKeysHoldFansAndTheCablesPort(t *testing.T) {
	kinds := map[string]Kind{}
	for _, k := range HostKeys() {
		kinds[k.Name] = k.Kind
	}
	if kinds["WS_FANS_INTERVAL"] != Decimal || kinds["WS_CABLE_PORT"] != Text {
		t.Errorf("the host's fans interval is %v, its cable's port %v; want a number and text", kinds["WS_FANS_INTERVAL"], kinds["WS_CABLE_PORT"])
	}
	for _, table := range []string{"WS_FANS_CURVES", "WS_FANS_STEP"} {
		if _, ok := kinds[table]; ok {
			t.Errorf("key %q is a table", table)
		}
	}
}

// A fraction is one key's value alone: the fans' interval.
func TestAnIntervalTakesAFractionAndNoTextOrInfinity(t *testing.T) {
	var interval Key
	for _, k := range HostKeys() {
		if k.Name == "WS_FANS_INTERVAL" {
			interval = k
		}
	}
	for value, want := range map[string]string{"soon": "a number", "NaN": "a number", "Inf": "a number", "2.5": ""} {
		err := interval.Check(value)
		if want == "" && err != nil || want != "" && (err == nil || !strings.Contains(err.Error(), want)) {
			t.Errorf("%q: err = %v, want %q", value, err, want)
		}
	}
	if got, err := interval.line("1.50"); err != nil || got != "WS_FANS_INTERVAL: 1.5\n" {
		t.Errorf("line = %q, %v, want WS_FANS_INTERVAL: 1.5", got, err)
	}
	h, err := ParseHost([]byte("WS_FANS_INTERVAL: 1.5\n"))
	if err != nil || h.Text(interval) != "1.5" {
		t.Errorf("the interval reads %q, %v; want 1.5", h.Text(interval), err)
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
