package tunnel

import (
	"reflect"
	"testing"
	"time"
)

func TestStateRoundTrip(t *testing.T) {
	s := State{Tier: TierLAN, Endpoint: "192.168.1.10:51820", Interface: "utun4",
		Since: time.Unix(1_761_900_000, 0), Routes: nets("10.1.0.0/16", "172.16.0.0/12"), LeftOut: nets("192.168.1.0/24")}
	want := "tier=lan\nendpoint=192.168.1.10:51820\ninterface=utun4\nsince=1761900000\n" +
		"routes=10.1.0.0/16 172.16.0.0/12\nleft_out=192.168.1.0/24\n"
	if got := s.Encode(); got != want {
		t.Fatalf("encoded %q, want %q", got, want)
	}
	got, err := ParseState(want)
	if err != nil || !reflect.DeepEqual(got, s) {
		t.Errorf("parsed %+v, %v; want %+v", got, err, s)
	}
}

func TestParseStateErrors(t *testing.T) {
	for _, in := range []string{"since=soon\n", "routes=10.1.0.0/16 nope\n", "left_out=10.1.0.5\n"} {
		if _, err := ParseState(in); err == nil {
			t.Errorf("%q: no error", in)
		}
	}
}
