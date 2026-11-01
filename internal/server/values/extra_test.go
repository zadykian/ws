package values

import "testing"

func TestOverrideSetsTheKeysOfValues(t *testing.T) {
	in := "security_outbound: true\nbase_swappiness: 10\n"
	data, err := Override([]byte(in), []string{"security_outbound=false", "base_swap_size_mb=64", "ws_user=bob", "nothing=1", "bare"})
	if err != nil {
		t.Fatal(err)
	}
	v, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse(%q): %v", data, err)
	}
	if v.Security.Outbound || v.Base.SwapSizeMB == nil || *v.Base.SwapSizeMB != 64 || v.Base.Swappiness != 10 || v.User != "bob" {
		t.Errorf("got %+v from %q", v, data)
	}
}

func TestOverrideOfNoFile(t *testing.T) {
	data, err := Override(nil, []string{"security_outbound=false"})
	if err != nil {
		t.Fatal(err)
	}
	if v, err := Parse(data); err != nil || v.Security.Outbound {
		t.Errorf("Parse(%q) = %v, %v", data, v.Security.Outbound, err)
	}
}
