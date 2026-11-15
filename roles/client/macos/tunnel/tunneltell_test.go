package tunnel_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
)

func TestTunnelWritesThePublicKeyWhereTheWizardReadsIt(t *testing.T) {
	m := daemonMachine()
	delete(m.Files, "/etc/ws/tunnel.pub")
	got := lines(tunnelChanges(t, daemonStep(m)))
	if !strings.HasPrefix(got, "create /etc/ws/tunnel.pub\n") || !strings.HasSuffix(got, "+PUB\n") {
		t.Errorf("Plan =\n%s\nwant /etc/ws/tunnel.pub, root's", got)
	}
}

func TestTunnelFinishPrintsNothingOnceTheDaemonHasItsPeer(t *testing.T) {
	var text bytes.Buffer
	s := daemonStep(daemonMachine())
	s.Text = &text
	if err := s.Finish(context.Background()); err != nil || text.Len() != 0 {
		t.Errorf("Finish = %v, %q; want nothing", err, text.String())
	}
}

func TestTunnelFinishPrintsTheKeyWhereMacConfLacksWsKey(t *testing.T) {
	var text bytes.Buffer
	m := daemonMachine()
	s := daemonStep(m)
	s.Conf.PublicKey, s.Conf.User, s.Text = "", "ws", &text
	if err := s.Finish(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"public key is PUB\nMake it ws's peer, mac, with a direct way to ws, as ssh ws needs", "ssh -t -o HostName=ADDRESS -o Port=PORT ws \"printf '%s\\n' 'PUB' | sudo tee /etc/wireguard/mac.pub",
		"ssh -t -o HostName=ADDRESS -o Port=PORT ws sudo ws server ubuntu apply --only tunnel\n", "Then set WS_PUBLIC_KEY"} {
		if !strings.Contains(text.String(), want) {
			t.Errorf("Finish printed %q; want %q in it", text.String(), want)
		}
	}
}

// A server not named ws is the commands' host, as config.d/ws names its Host after it, and the
// Mac becomes its peer.
func TestTunnelFinishNamesTheServerAsTheHost(t *testing.T) {
	var text bytes.Buffer
	s := daemonStep(daemonMachine())
	s.Server, s.Conf.PublicKey, s.Conf.User, s.Text = "box", "", "root", &text
	if err := s.Finish(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := "The Mac's WireGuard public key is PUB\nMake it box's peer, mac, with a direct way to box, " +
		"as ssh box needs this done first. ADDRESS and PORT are that way:\n\n" +
		"    ssh -o HostName=ADDRESS -o Port=PORT box \"printf '%s\\n' 'PUB' | tee /etc/wireguard/mac.pub >/dev/null\"\n" +
		"    ssh -o HostName=ADDRESS -o Port=PORT box ws server ubuntu apply --only tunnel\n" +
		"\nThen set WS_PUBLIC_KEY in /etc/ws/servers/NAME/values.yml to box's key, which that run prints, and run\n" +
		"ws client macos apply again, which installs the tunnel's daemon.\n"
	if text.String() != want {
		t.Errorf("Finish printed\n%s\nwant\n%s", text.String(), want)
	}
}

func TestTunnelFinishPrintsNothingUnderCheck(t *testing.T) {
	var text bytes.Buffer
	s := daemonStep(daemonMachine())
	s.Conf.PublicKey, s.Check, s.Text = "", true, &text
	if err := s.Finish(context.Background()); err != nil || text.Len() != 0 {
		t.Errorf("Finish = %v, %q; want nothing", err, text.String())
	}
}

func TestTunnelDoesNotRunRootsCopyWhereItOrItsDirectoryIsToChange(t *testing.T) {
	m := daemonMachine()
	m.Root["/usr/local/libexec/ws-tunnel"] = effect.Attrs{UID: 501, GID: 20, Mode: 0o777}
	delete(m.Files, "/etc/ws/tunnel.pub")
	tunnelChanges(t, daemonStep(m))
	for _, c := range m.Calls {
		if strings.Contains(c, "ws client macos tunnel key") {
			t.Errorf("calls = %q; want root's copy not run before Apply fixes it", m.Calls)
		}
	}
}

func TestTunnelReportsThePublicKeyAsRootsOnceItHoldsTheKey(t *testing.T) {
	m := daemonMachine()
	s := daemonStep(m)
	tunnelChanges(t, s)
	items, err := s.Report(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.Name == "/etc/ws/tunnel.pub" && it.Want == "owner 0:0, mode 644" {
			return
		}
	}
	t.Errorf("Report = %+v, want /etc/ws/tunnel.pub at owner 0:0, mode 644", items)
}
