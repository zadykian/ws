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
		"ws_user": Text, "base_swappiness": Number, "base_swap_size_mb": Number, "fans_interval": Decimal,
		"security_outbound": Switch, "ikeward_domains": List, "zed_nofile": Number,
	} {
		if got, ok := kinds[name]; !ok || got != want {
			t.Errorf("%s: kind = %v, %t, want %v", name, got, ok, want)
		}
	}
	for _, table := range []string{"fans_curves", "fans_step", "claude_telemetry_env", "wireguard_peers", ""} {
		if _, ok := kinds[table]; ok {
			t.Errorf("key %q is a table, or has no name", table)
		}
	}
}

func TestCheckRefusesWhatTheKindCantTake(t *testing.T) {
	for _, tc := range []struct {
		key, value, want string
	}{
		{"base_swappiness", "ten", "a whole number"},
		{"base_swappiness", "1.5", "a whole number"},
		{"fans_interval", "soon", "a number"},
		{"fans_interval", "NaN", "a number"},
		{"fans_interval", "Inf", "a number"},
		{"security_outbound", "yes", "true or false"},
		{"ws_user", "a\nb", "one line"},
		{"ikeward_domains", "a\rb", "one line"},
		{"base_swappiness", "20", ""},
		{"base_swappiness", "", ""},
		{"fans_interval", "2.5", ""},
		{"security_outbound", "false", ""},
		{"ws_user", "ada", ""},
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
