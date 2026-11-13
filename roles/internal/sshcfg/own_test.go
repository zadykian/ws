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
