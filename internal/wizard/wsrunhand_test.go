package wizard

import (
	"context"
	"strings"
	"testing"
)

func TestWsRunHand(t *testing.T) {
	for name, tc := range map[string]struct{ target, status, want string }{
		"lan":   {"lan", "fans refused\n", "ws's fans' driver"},
		"local": {"lan", "", "macOS's local network"},
		"cloud": {"cloud", "outbound off\n", "The provider's firewall: UDP 51820"},
		"off":   {"lan", "outbound off\n", "ws's outbound table"},
	} {
		r, _ := newRun(tc.target, tc.status, false, false)
		h := &Hand{}
		r.Hand(context.Background(), h)
		if got := strings.Join(h.lines, "\n"); !strings.Contains(got, tc.want) {
			t.Errorf("%s: Hand = %q; want %q", name, got, tc.want)
		}
	}
}

// ws's own list holds ws's lines alone, with nothing of the Mac's.
func TestWsRunWsHand(t *testing.T) {
	for status, want := range map[string]string{"fans refused\n": "ws's fans' driver", "fans ok\n": ""} {
		r, _ := newRun("lan", status, false, false)
		h := &Hand{}
		r.WsHand(context.Background(), h)
		if got := strings.Join(h.lines, "\n"); (want == "") != (got == "") || !strings.HasPrefix(got, want) {
			t.Errorf("%q: WsHand = %q; want %q", status, got, want)
		}
	}
}
