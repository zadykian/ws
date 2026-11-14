package host

import (
	"context"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/remote"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"strings"
	"testing"
)

func TestMacHand(t *testing.T) {
	for name, tc := range map[string]struct {
		target string
		status session.Status
		want   string
	}{
		"lan":   {"lan", session.Status{FansRefused: true}, "ws's fans' driver"},
		"local": {"lan", session.Status{}, "macOS's local network"},
		"cloud": {"cloud", session.Status{OutboundOff: true}, "The provider's firewall: UDP 51820"},
		"off":   {"lan", session.Status{OutboundOff: true}, "ws's outbound table"},
	} {
		r, _ := newRun(tc.target, tc.status, false, false)
		h := &guided.Hand{}
		r.Ws.Hand(context.Background(), h)
		if got := strings.Join(h.Lines(), "\n"); !strings.Contains(got, tc.want) {
			t.Errorf("%s: Hand = %q; want %q", name, got, tc.want)
		}
	}
}

func TestMacHandNamesTheServer(t *testing.T) {
	for name, tc := range map[string]struct {
		target string
		want   []string
	}{
		"lan": {"lan", []string{"lab's lease: static, as", "reached lab at 10.0.0.5.", "The router's forwards to lab, UDP 51820"}},
		"cloud": {"cloud", []string{"lab's address: WS_REMOTE in server.conf must lead to lab, as this run reached it at 10.0.0.5",
			"UDP 51820 open to lab, for"}},
	} {
		m := effecttest.NewMachine()
		m.Files["/h/mac.conf"] = []byte("WS_TARGET=" + tc.target + "\nWS_USER=ada\n" + extra["cloud"])
		m.Files["/l/"+remote.FirstFile] = []byte("10.0.0.5\n")
		r := MacReach{SSH: remote.SSH{Host: "lab", M: m, Lock: "/l"}, Conf: "/h/mac.conf", Out: &effecttest.Out{}}
		h := &guided.Hand{}
		r.Hand(h, session.Status{})
		got := strings.Join(h.Lines(), "\n")
		for _, w := range tc.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: Hand = %q; want %q", name, got, w)
			}
		}
	}
}
