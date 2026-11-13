package host

import (
	"context"
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
