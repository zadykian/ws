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
		{nil, "", "", ""},
		{nil, "lab", "lab", ""},
		{[]string{"box"}, "", "box", ""},
		{[]string{"box"}, "box", "box", ""},
		{[]string{"box"}, "lab", "lab", ""},
		{[]string{"box", "ws"}, "lab", "lab", ""},
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

func TestWizardServerRefusesANewNameThatSSHConfigHolds(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/.ssh/config"] = []byte("Host a\n  User x\nHost box lab\n  User y\n")
	m.Files[conf.ServerPath("/h/.config/ws/mac.conf", "box")] = nil
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = withServerDirs(m)
	want := "line 3 of ~/.ssh/config, Host box lab, sets its own values for lab: pick another name, or remove it"
	if got, err := wizardServer(p, "lab"); got != "" || err == nil || err.Error() != want {
		t.Errorf("wizardServer(lab) = %q, %v; want the refusal %q", got, err, want)
	}
	if got, err := wizardServer(p, "box"); got != "box" || err != nil {
		t.Errorf("wizardServer(box) = %q, %v; want the server that is set up already", got, err)
	}
}
