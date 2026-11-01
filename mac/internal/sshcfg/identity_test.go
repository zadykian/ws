package sshcfg_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/sshcfg"
)

const home = "/Users/me"

func TestPickIdentity(t *testing.T) {
	data := "~/" + sshcfg.SecretiveData
	tests := []struct {
		name, identity, agent string
		socket                bool
		keys                  []string
		want                  sshcfg.Identity
	}{
		{"the file key", "", "", false, nil, sshcfg.Identity{Pub: sshcfg.DefaultPub}},
		{"no socket, so the file key", "", "", false, []string{"a.pub"}, sshcfg.Identity{Pub: sshcfg.DefaultPub}},
		{"mac.conf's key", "~/k.pub", "", true, []string{"a.pub"}, sshcfg.Identity{Pub: "~/k.pub", Check: true}},
		{"Secretive's one key", "", "", true, []string{"a.pub", "a-cert.pub", "sign.pub", "b c.pub", ".d.pub", "x"},
			sshcfg.Identity{Pub: data + "/PublicKeys/a.pub", Agent: data + "/socket.ssh", Check: true}},
	}
	for _, tt := range tests {
		got, err := sshcfg.PickIdentity(tt.identity, tt.agent, "~/Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data/PublicKeys/sign.pub",
			home, tt.socket, tt.keys)
		if err != nil || got != tt.want {
			t.Errorf("%s: PickIdentity = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
	_, err := sshcfg.PickIdentity("", "", "", home, true, []string{"a.pub", "b.pub"})
	if err == nil || !strings.Contains(err.Error(), "several keys") {
		t.Errorf("two keys: err = %v", err)
	}
}
