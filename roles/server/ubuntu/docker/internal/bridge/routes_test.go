package bridge

import (
	"testing"
)

func TestParseRoutesReadsIPsJSON(t *testing.T) {
	got, err := parseRoutes([]byte(`[{"dst":"172.18.0.0/16","dev":"br-1","scope":"link"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if want := (route{Dst: "172.18.0.0/16", Dev: "br-1", Scope: "link"}); len(got) != 1 || got[0] != want {
		t.Errorf("parseRoutes = %+v, want %+v", got, want)
	}
	if _, err := parseRoutes([]byte("no json")); err == nil {
		t.Error("text that is no JSON passed")
	}
}
