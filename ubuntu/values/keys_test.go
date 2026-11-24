package values

import (
	"slices"
	"strings"
	"testing"
)

// keyOf returns the key of that name.
func keyOf(t *testing.T, name string) Key {
	keys := Keys()
	at := slices.IndexFunc(keys, func(k Key) bool { return k.Name == name })
	if at < 0 {
		t.Fatalf("no key %s", name)
	}
	return keys[at]
}

func TestKeysHoldEachKindAndNoTable(t *testing.T) {
	kinds := map[string]Kind{}
	for _, k := range Keys() {
		kinds[k.Name] = k.Kind
	}
	for name, want := range map[string]Kind{
		"WS_BASE_TIMEZONE": Text, "WS_BASE_SWAPPINESS": Number, "WS_BASE_SWAP_SIZE_MB": Number,
		"WS_IKEWARD_DOMAINS": List, "WS_ZED_NOFILE": Number,
	} {
		if got, ok := kinds[name]; !ok || got != want {
			t.Errorf("%s: kind = %v, %t, want %v", name, got, ok, want)
		}
	}
	for _, table := range []string{"WS_CLAUDE_TELEMETRY_ENV", "WS_TUNNEL_PEERS", ""} {
		if _, ok := kinds[table]; ok {
			t.Errorf("key %q is a table, or has no name", table)
		}
	}
}

func TestCheckRefusesWhatTheKindCantTake(t *testing.T) {
	for _, tc := range []struct {
		key, value, want string
	}{
		{"WS_BASE_SWAPPINESS", "ten", "a whole number"},
		{"WS_BASE_SWAPPINESS", "1.5", "a whole number"},
		{"WS_BASE_TIMEZONE", "a\nb", "one line"},
		{"WS_IKEWARD_DOMAINS", "a\rb", "one line"},
		{"WS_BASE_SWAPPINESS", "20", ""},
		{"WS_BASE_SWAPPINESS", "", ""},
		{"WS_BASE_TIMEZONE", "Asia/Tbilisi", ""},
	} {
		err := keyOf(t, tc.key).Check(tc.value)
		if tc.want == "" && err != nil {
			t.Errorf("%s %q: %v", tc.key, tc.value, err)
		}
		if tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Errorf("%s %q: err = %v, want %q", tc.key, tc.value, err, tc.want)
		}
	}
}

func TestTextGivesTheValueAsALine(t *testing.T) {
	v, err := Parse([]byte("WS_IKEWARD_DOMAINS: [a.example, b.example]\n"))
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"WS_IKEWARD_DOMAINS": "a.example, b.example",
		"WS_BASE_SWAPPINESS": "10", "WS_BASE_SWAP_SIZE_MB": "", "WS_GIT_SIGNING_PUB_KEY": "/root/.ssh/id_ed25519_signing.pub", "WS_IKEWARD_DNS": "",
	} {
		if got := v.Text(keyOf(t, name)); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	size := 0
	v.Base.SwapSizeMB = &size
	if got := v.Text(keyOf(t, "WS_BASE_SWAP_SIZE_MB")); got != "0" {
		t.Errorf("a swap of 0 reads %q", got)
	}
}

func TestLineQuotesWhatReadsAsAnotherType(t *testing.T) {
	for _, tc := range []struct {
		key, value, want string
	}{
		{"WS_DOCKER_MEMORY_SWAP_MAX", "0", "WS_DOCKER_MEMORY_SWAP_MAX: \"0\"\n"},
		{"WS_BASE_TIMEZONE", "a #b", "WS_BASE_TIMEZONE: 'a #b'\n"},
		{"WS_IKEWARD_DOMAINS", ",", "WS_IKEWARD_DOMAINS: []\n"},
		{"WS_IKEWARD_DOMAINS", "a.example , b.example", "WS_IKEWARD_DOMAINS: [a.example, b.example]\n"},
		{"WS_BASE_SWAPPINESS", "20", "WS_BASE_SWAPPINESS: 20\n"},
	} {
		if got, err := keyOf(t, tc.key).line(tc.value); err != nil || got != tc.want {
			t.Errorf("%s %q = %q, %v, want %q", tc.key, tc.value, got, err, tc.want)
		}
	}
}

func TestLineKeepsALongValueOnOneLine(t *testing.T) {
	long := "/root/" + strings.Repeat("word ", 60) + "end.pub"
	line, err := keyOf(t, "WS_GIT_SIGNING_PUB_KEY").line(long)
	if err != nil {
		t.Fatal(err)
	}
	if v, err := Parse([]byte(line)); err != nil || v.GitSigningPubKey != long {
		t.Errorf("reads %q, %v", v.GitSigningPubKey, err)
	}
}

// No key is a switch yet: a role's next one is a bool, which the kind takes whatever its name.
func TestSwitchTakesTrueOrFalse(t *testing.T) {
	k := Key{Name: "WS_SWITCH", Kind: Switch}
	for value, want := range map[string]string{"true": "WS_SWITCH: true\n", "false": "WS_SWITCH: false\n"} {
		if line, err := k.line(value); err != nil || line != want || k.Check(value) != nil {
			t.Errorf("%q: line %q, %v; want %q", value, line, err, want)
		}
	}
	if err := k.Check("yes"); err == nil || !strings.Contains(err.Error(), "true or false") {
		t.Errorf("yes: err = %v, want the refusal", err)
	}
}
