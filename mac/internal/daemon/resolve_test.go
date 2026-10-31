package daemon

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func TestResolve(t *testing.T) {
	const query = "dscacheutil -q host -a name ws.example.org"
	tests := []struct {
		name   string
		host   string
		result effecttest.Result
		want   netip.Addr
	}{
		{"address", "192.0.2.7", effecttest.Result{}, netip.MustParseAddr("192.0.2.7")},
		{"ipv6 address", "2001:db8::1", effecttest.Result{}, netip.Addr{}},
		{"first answer", "ws.example.org",
			effecttest.Result{Out: []byte("name: ws.example.org\nip_address: 192.0.2.9\nip_address: 192.0.2.8\n")}, netip.MustParseAddr("192.0.2.9")},
		{"ipv6 first", "ws.example.org",
			effecttest.Result{Out: []byte("name: ws.example.org\nip_address: 2001:db8::1\nip_address: 192.0.2.8\n")}, netip.Addr{}},
		{"no answer", "ws.example.org", effecttest.Result{}, netip.Addr{}},
		{"fails", "ws.example.org", effecttest.Result{Err: errors.New("no")}, netip.Addr{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := effecttest.NewMachine()
			m.Results[query] = tt.result
			got := Resolve(context.Background(), m, tt.host)
			if got != tt.want {
				t.Errorf("Resolve = %v, want %v", got, tt.want)
			}
		})
	}
}
