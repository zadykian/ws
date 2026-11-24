package object

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func decode(t *testing.T, text string) map[string]any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		t.Fatal(err)
	}
	return plain(m).(map[string]any)
}

func TestDiffersNamesTheDeclaredKeysThatDiffer(t *testing.T) {
	have := decode(t, `{"config":{"a":"1","b":"2","c":"x"},"devices":{"eth0":{"type":"nic","name":"eth0"}},"extra":1}`)
	for _, tc := range []struct {
		name string
		want map[string]any
		keys []string
	}{
		{"keys of its own are no difference", map[string]any{"config": map[string]any{"a": "1"}}, nil},
		{"a value that differs", map[string]any{"config": map[string]any{"a": "9", "b": "2"}}, []string{"config.a"}},
		{"a key that is missing", map[string]any{"config": map[string]any{"z": "1"}}, []string{"config.z"}},
		{"a device's key", map[string]any{"devices": map[string]any{"eth0": map[string]any{"name": "eth1"}}},
			[]string{"devices.eth0.name"}},
		{"a missing device", map[string]any{"devices": map[string]any{"root": map[string]any{"type": "disk"}}},
			[]string{"devices.root"}},
		{"an empty map that is missing", map[string]any{"limits": map[string]any{}}, nil},
		{"a key that is no map", map[string]any{"extra": map[string]any{"a": "1"}}, []string{"extra"}},
		{"a number as text", map[string]any{"extra": "1"}, nil},
		{"a device declared absent that is there", map[string]any{"devices": map[string]any{"eth0": nil}}, []string{"devices.eth0"}},
		{"a device declared absent that is missing", map[string]any{"devices": map[string]any{"gpu": nil}}, nil},
	} {
		if got := differs(tc.want, have, ""); !slices.Equal(got, tc.keys) {
			t.Errorf("%s: differs = %q, want %q", tc.name, got, tc.keys)
		}
	}
}

func TestDiffersComparesAListWhole(t *testing.T) {
	have := decode(t, `{"egress":[{"action":"drop","state":"enabled","description":"d"},{"action":"allow"}]}`)
	rule := func(action string) any { return map[string]any{"action": action} }
	for _, tc := range []struct {
		name string
		want []any
		keys []string
	}{
		{"the same rules, with a key that incus adds", []any{rule("drop"), rule("allow")}, nil},
		{"a rule that differs", []any{rule("drop"), rule("reject")}, []string{"egress"}},
		{"a rule less", []any{rule("drop")}, []string{"egress"}},
		{"a rule more", []any{rule("drop"), rule("allow"), rule("drop")}, []string{"egress"}},
	} {
		if got := differs(map[string]any{"egress": tc.want}, have, ""); !slices.Equal(got, tc.keys) {
			t.Errorf("%s: differs = %q, want %q", tc.name, got, tc.keys)
		}
	}
}

func TestMergeLaysTheDeclaredKeysOnTheBase(t *testing.T) {
	base := map[string]any{"config": map[string]any{"a": "1", "volatile.x": "9"}, "ingress": []any{"old"}}
	over := map[string]any{"config": map[string]any{"a": "2", "b": "3"}, "ingress": []any{"new"}}
	want := map[string]any{"config": map[string]any{"a": "2", "b": "3", "volatile.x": "9"}, "ingress": []any{"new"}}
	if got := merge(base, over); !reflect.DeepEqual(got, want) {
		t.Errorf("merge = %v, want %v", got, want)
	}
	if base["config"].(map[string]any)["a"] != "1" {
		t.Errorf("merge changed its base: %v", base)
	}
}

func TestMergeRemovesAKeyDeclaredAbsent(t *testing.T) {
	base := map[string]any{"devices": map[string]any{"eth0": map[string]any{"type": "nic"}, "gpu": map[string]any{"type": "gpu"}}}
	over := map[string]any{"devices": map[string]any{"gpu": nil, "other": nil}}
	want := map[string]any{"devices": map[string]any{"eth0": map[string]any{"type": "nic"}}}
	if got := merge(base, over); !reflect.DeepEqual(got, want) {
		t.Errorf("merge = %v, want %v", got, want)
	}
	if got := merge(nil, over); !reflect.DeepEqual(got, map[string]any{"devices": map[string]any{}}) {
		t.Errorf("merge into nothing = %v, want no key of nil", got)
	}
	if _, kept := base["devices"].(map[string]any)["gpu"]; !kept {
		t.Errorf("merge changed its base: %v", base)
	}
}

func TestPickKeepsTheKeysThatExist(t *testing.T) {
	got := pick(map[string]any{"config": 1, "used_by": 2}, []string{"config", "description"})
	if !reflect.DeepEqual(got, map[string]any{"config": 1}) {
		t.Errorf("pick = %v", got)
	}
}

func TestPlainTurnsJSONNumbersIntoNumbers(t *testing.T) {
	got := decode(t, `{"n":1000000,"f":1.5,"l":[2]}`)
	want := map[string]any{"n": int64(1000000), "f": 1.5, "l": []any{int64(2)}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("plain = %v, want %v", got, want)
	}
}
