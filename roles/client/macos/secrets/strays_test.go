package secrets_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/roles/client/macos/secrets"
)

func TestStraysAreTheHostKeysThatKnownHostsLacks(t *testing.T) {
	f, _, err := secrets.SetHostKey(repository(), "ws-b", otherHost)
	if err != nil {
		t.Fatal(err)
	}
	keys := []secrets.DeployKey{{ID: 1, Title: "ws-x", Key: hostKey, ReadOnly: true},
		{ID: 2, Title: "ws-old", Key: newerKey, ReadOnly: true},
		{ID: 3, Title: "ci", Key: "ssh-rsa AAAAB3NzaC1yc2E", ReadOnly: true}}
	strays, err := secrets.Strays(f, keys, map[string]bool{otherHost: true})
	want := []secrets.Stray{{Key: hostKey, Names: []string{"ws-a", "ws-x"}}, {Key: newerKey, Names: []string{"ws-old"}}}
	same := func(a, b secrets.Stray) bool { return a.Key == b.Key && slices.Equal(a.Names, b.Names) }
	if err != nil || !slices.EqualFunc(strays, want, same) {
		t.Errorf("Strays gives %v, %v, want %v", strays, err, want)
	}
	known := map[string]bool{hostKey: true, otherHost: true, newerKey: true}
	if strays, err := secrets.Strays(f, keys, known); err != nil || len(strays) != 0 {
		t.Errorf("Strays where known_hosts holds every key gives %v, %v, want none", strays, err)
	}
}

func TestStraysFailOnATableOfAnotherForm(t *testing.T) {
	f := repository()
	f[secrets.SharedFile] = append(f[secrets.SharedFile], "\n[providers.age.recipients.more]\n"...)
	if _, err := secrets.Strays(f, nil, nil); err == nil {
		t.Error("Strays reads a table that ws cannot")
	}
}
