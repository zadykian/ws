package wizard

import (
	"context"
	"strings"
	"testing"
)

func TestMacHand(t *testing.T) {
	for name, tc := range map[string]struct {
		target string
		status wsStatus
		want   string
	}{
		"lan":   {"lan", wsStatus{FansRefused: true}, "ws's fans' driver"},
		"local": {"lan", wsStatus{}, "macOS's local network"},
		"cloud": {"cloud", wsStatus{OutboundOff: true}, "The provider's firewall: UDP 51820"},
		"off":   {"lan", wsStatus{OutboundOff: true}, "ws's outbound table"},
	} {
		r, _ := newRun(tc.target, tc.status, false, false)
		h := &Hand{}
		r.Ws.Hand(context.Background(), h)
		if got := strings.Join(h.lines, "\n"); !strings.Contains(got, tc.want) {
			t.Errorf("%s: Hand = %q; want %q", name, got, tc.want)
		}
	}
}
