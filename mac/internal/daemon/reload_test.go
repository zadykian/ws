package daemon

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/tunnel"
	"github.com/zadykian/ws/mac/internal/uapi"
)

func TestReloadChangesTheRoutesInPlace(t *testing.T) {
	r := onTheCable(t)
	r.m.Files[dir+"/routes.d/office"] = []byte("172.16.0.0/12\n")
	r.m.table["ls -1 "+dir+"/routes.d"] = effecttest.Result{Out: []byte("office\n")}
	r.reload(context.Background())
	if got := added(r); len(got) != 1 {
		t.Errorf("endpoints = %q, want the session kept", got)
	}
	want, _ := uapi.SetAllowed(peerKey, r.allowed.IPs())
	if got := r.dev.sets[len(r.dev.sets)-1]; got != want || !strings.Contains(want, "172.16.0.0/12") {
		t.Errorf("last set = %q, want AllowedIPs with the office %q", got, want)
	}
	if !slices.Contains(r.m.runs, "route -q -n add -inet 172.16.0.0/12 -interface utun4") {
		t.Errorf("commands = %q", r.m.runs)
	}
	if got := r.lines()[len(r.lines())-1]; got != "read its settings again; routes: 172.16.0.0/12" {
		t.Errorf("log = %q", r.lines())
	}
}

func TestReloadTriesAgainOnNewSettings(t *testing.T) {
	r := onTheCable(t)
	r.m.Files[dir+"/tunnel.conf"] = []byte("WS_PUBLIC_KEY=" + peerKey + "\n" +
		strings.Replace(confAll, "192.168.1.7", "192.168.1.8", 1))
	r.reload(context.Background())
	if got := added(r); !slices.Equal(got, []string{"10.77.0.1:51820", "10.77.0.1:51820"}) {
		t.Errorf("endpoints = %q, want the cable tried again", got)
	}
	if got := r.lines()[len(r.lines())-1]; got != "read new settings; trying the paths again" {
		t.Errorf("log = %q", r.lines())
	}
}

func TestReloadKeepsTheSettingsItHad(t *testing.T) {
	r := onTheCable(t)
	delete(r.m.Files, dir+"/tunnel.conf")
	r.reload(context.Background())
	if got := added(r); len(got) != 1 {
		t.Errorf("endpoints = %q, want none new", got)
	}
	if got := r.lines()[len(r.lines())-2]; !strings.HasSuffix(got, "keeps the settings it had") {
		t.Errorf("log = %q", r.lines())
	}
}

func TestReloadSetsAllowedIPsDuringTheFirstRound(t *testing.T) {
	r := started(t)
	ctx := context.Background()
	r.choose(ctx) // the round has tried the cable, and nothing is in use yet
	r.m.Files[dir+"/routes.d/office"] = []byte("172.16.0.0/12\n")
	r.m.table["ls -1 "+dir+"/routes.d"] = effecttest.Result{Out: []byte("office\n")}
	r.reload(ctx)
	if got := r.dev.sets[len(r.dev.sets)-1]; !strings.Contains(got, "172.16.0.0/12") {
		t.Errorf("last set = %q, want AllowedIPs with the office", got)
	}
}

func TestReloadRoutesTheNetworkOfAFailedTier(t *testing.T) {
	r := started(t)
	ctx := context.Background()
	r.choose(ctx) // the cable
	r.now = r.now.Add(tunnel.Patience)
	r.choose(ctx) // the LAN, which holds 192.168.1.7
	r.now = r.now.Add(tunnel.Patience)
	r.choose(ctx) // the name
	r.dev.get = peerText("203.0.113.9:51820", r.now)
	r.choose(ctx) // the name answers
	r.m.Files[dir+"/routes.d/home"] = []byte("192.168.1.0/24\n")
	r.m.table["ls -1 "+dir+"/routes.d"] = effecttest.Result{Out: []byte("home\n")}
	r.reload(ctx)
	if got := r.dev.sets[len(r.dev.sets)-1]; !strings.Contains(got, "192.168.1.0/24") {
		t.Errorf("last set = %q, want AllowedIPs with the LAN, as the endpoint in use is the name", got)
	}
}
