package daemon

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/wgkey"
)

const stateText = "tier=cable\nendpoint=10.77.0.1:51820\ninterface=utun4\nsince=1793448000\n" +
	"routes=172.16.0.0/12\nleft_out=\n"

func statusMachine(state string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Files[dir+"/tunnel.conf"] = []byte("WS_PUBLIC_KEY=" + peerKey + "\n")
	if state != "" {
		m.Files["/var/run/ws-tunnel.state"] = []byte(state)
	}
	return m
}

func TestStatus(t *testing.T) {
	hexKey, _ := wgkey.Hex(peerKey)
	peer := "public_key=" + hexKey + "\nlast_handshake_time_sec=" + "1793447990" +
		"\nlast_handshake_time_nsec=0\nrx_bytes=100\ntx_bytes=200\n"
	now := time.Unix(1793448010, 0)
	tests := []struct {
		name string
		get  GetFunc
		want string
	}{
		{"answered", func(context.Context, string) (string, error) { return peer, nil },
			"tier: cable\nendpoint: 10.77.0.1:51820\ninterface: utun4\nsince: 10 s ago\n" +
				"latest handshake: 20 s ago\ntransfer: 100 B received, 200 B sent\nroutes: 172.16.0.0/12\n"},
		{"no device", func(context.Context, string) (string, error) { return "", errTest },
			"tier: cable\nendpoint: 10.77.0.1:51820\ninterface: utun4\nsince: 10 s ago\n" +
				"latest handshake: none\nroutes: 172.16.0.0/12\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Status(context.Background(), statusMachine(stateText), tt.get, dir,
				"/var/run/ws-tunnel.state", now)
			if err != nil || got != tt.want {
				t.Errorf("Status = %q, %v, want %q", got, err, tt.want)
			}
		})
	}
}

func TestStatusSocket(t *testing.T) {
	var sock string
	get := func(_ context.Context, s string) (string, error) { sock = s; return "", nil }
	_, _ = Status(context.Background(), statusMachine(stateText), get, dir, "/var/run/ws-tunnel.state", t0)
	if sock != "/var/run/wireguard/utun4.sock" {
		t.Errorf("socket = %q", sock)
	}
}

func TestStatusFails(t *testing.T) {
	get := func(context.Context, string) (string, error) { return "", nil }
	noConf := statusMachine(stateText)
	delete(noConf.Files, dir+"/tunnel.conf")
	tests := []struct {
		name string
		m    *effecttest.Machine
		want string
	}{
		{"not running", statusMachine(""), "ws tunnel isn't running; its log is /var/log/ws-tunnel.log"},
		{"bad state", statusMachine("since=x\n"), "state: since is 'x'"},
		{"no conf", noConf, "run mac/setup.sh"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Status(context.Background(), tt.m, get, dir, "/var/run/ws-tunnel.state", t0)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want one with %q", err, tt.want)
			}
		})
	}
}
