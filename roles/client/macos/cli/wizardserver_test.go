package cli

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

func TestWizardServerNamesTheServerToSetUp(t *testing.T) {
	const servers = "/h/.config/ws/servers"
	for _, tc := range []struct {
		have []string
		flag string
		want string
		err  string
	}{
		{nil, "", "ws", ""},
		{nil, "lab", "lab", ""},
		{[]string{"box"}, "", "box", ""},
		{[]string{"box"}, "box", "box", ""},
		{[]string{"box"}, "lab", "", "lab would be a second server, beside box: the wizard sets up one server"},
		{[]string{"box", "ws"}, "lab", "", "lab would be a second server, beside box and ws: the wizard sets up one server"},
		{[]string{"box", "ws"}, "", "", "several servers in " + servers + ", box and ws: name one by --server"},
		{nil, "Lab", "", "--server Lab is not a server's name: " + conf.NameForm},
	} {
		m := effecttest.NewMachine()
		for _, name := range tc.have {
			m.Files[conf.ServerPath("/h/.config/ws/mac.conf", name)] = nil
		}
		p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
		p.Machine = withServerDirs(m)
		got, err := wizardServer(p, tc.flag)
		if got != tc.want || (err == nil) != (tc.err == "") || (err != nil && !strings.Contains(err.Error(), tc.err)) {
			t.Errorf("wizardServer(%v, %q) = %q, %v; want %q, %q", tc.have, tc.flag, got, err, tc.want, tc.err)
		}
	}
}
