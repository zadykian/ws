package values

import "testing"

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
