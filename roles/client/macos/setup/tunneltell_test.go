package setup_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
)

func TestTunnelWritesThePublicKeyWhereTheWizardReadsIt(t *testing.T) {
	m := daemonMachine()
	delete(m.Files, "/h/.config/ws/tunnel.pub")
	delete(m.Dirs, "/h/.config/ws")
	got := lines(tunnelChanges(t, daemonStep(m)))
	if !strings.Contains(got, "would make /h/.config/ws, mode 700\n") || !strings.HasSuffix(got, "+PUB\n") {
		t.Errorf("Plan =\n%s\nwant the directory and tunnel.pub", got)
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
	for _, want := range []string{"public key is PUB\n", "ssh -t -o HostName=ADDRESS -o Port=PORT ws \"printf '%s\\n' 'PUB' | sudo tee /etc/wireguard/mac.pub",
		"ssh -t -o HostName=ADDRESS -o Port=PORT ws sudo ws server ubuntu apply --only wireguard\n", "Then set WS_PUBLIC_KEY"} {
		if !strings.Contains(text.String(), want) {
			t.Errorf("Finish printed %q; want %q in it", text.String(), want)
		}
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
	delete(m.Files, "/h/.config/ws/tunnel.pub")
	tunnelChanges(t, daemonStep(m))
	for _, c := range m.Calls {
		if strings.Contains(c, "ws client macos tunnel key") {
			t.Errorf("calls = %q; want root's copy not run before Apply fixes it", m.Calls)
		}
	}
}

func TestTunnelReportsWsDirectoryOnceItHoldsTheKey(t *testing.T) {
	m := daemonMachine()
	s := daemonStep(m)
	tunnelChanges(t, s)
	items, err := s.Report(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.Name == "/h/.config/ws" && it.Want == "mode 700" {
			return
		}
	}
	t.Errorf("Report = %+v, want /h/.config/ws at mode 700", items)
}
