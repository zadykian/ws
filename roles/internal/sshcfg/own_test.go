package sshcfg_test

import (
	"testing"

	"github.com/zadykian/ws/roles/internal/sshcfg"
)

func TestOwnHosts(t *testing.T) {
	text := "Host a\nhost ws ws-other ws\n  Match originalhost ws\n"
	got := sshcfg.OwnHosts(text, "ws", "ws-other")
	if len(got) != 2 || got[0].Line != 2 || got[0].Names != "ws and ws-other" || got[1].Line != 3 || got[1].Names != "ws" {
		t.Errorf("OwnHosts = %+v", got)
	}
}

func TestOwnHostsReadsAMatchLineByItsHostCriteria(t *testing.T) {
	text := "Match host github.com\nMatch user host exec host\nMatch all\nMatch originalhost a,host\nMatch final host host\n"
	got := sshcfg.OwnHosts(text, "host", "all", "user", "exec")
	if len(got) != 2 || got[0].Line != 4 || got[0].Names != "host" || got[1].Line != 5 || got[1].Names != "host" {
		t.Errorf("OwnHosts = %+v, want the lines 4 and 5 alone", got)
	}
	if got := sshcfg.OwnHosts("Host host all\n", "host"); len(got) != 1 {
		t.Errorf("OwnHosts = %+v, want the Host line", got)
	}
}
