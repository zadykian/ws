package wireguard

import (
	"slices"
	"strings"
	"testing"
)

func TestMacInterfacesLeaveOutAnUnnamedCable(t *testing.T) {
	if got := MacInterfaces(""); !slices.Equal(got, []string{"wg0"}) {
		t.Errorf("MacInterfaces(\"\") = %q", got)
	}
}

func TestDropsScriptIsExecutableShell(t *testing.T) {
	got, err := DropsScript()
	if err != nil || !strings.HasPrefix(string(got), "#!/bin/sh\n") {
		t.Errorf("DropsScript = %.20q, %v", got, err)
	}
}
