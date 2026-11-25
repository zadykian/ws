package incus

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/ubuntu/values"
)

// sourcesTable fills the table of sources for two guests, lab in slot 1 and build in none.
func sourcesTable(t *testing.T) string {
	t.Helper()
	m := effecttest.NewMachine()
	m.Files["/etc/ws/values.yml"] = []byte("WS_INCUS_GUESTS:\n" +
		"  - {name: lab, address: 10.100.0.10, cpu: 4, memory: 8GiB, disk: 200GiB}\n" +
		"  - {name: build, address: 10.100.0.11, cpu: 2, memory: 4GiB, disk: 100GiB}\n")
	m.Files["/etc/ws/guests/lab/values.yml"] = []byte("WS_TUNNEL_ADDRESS: 10.99.1.1/24\n")
	h, err := values.LoadHost(m, "/etc/ws/values.yml")
	if err != nil {
		t.Fatal(err)
	}
	table, err := resource.Render(os.DirFS("files"), "sources.nft.tmpl", h)
	if err != nil {
		t.Fatal(err)
	}
	return string(table)
}

// guestChain returns the rules of the table's chain from_guest, a line each.
func guestChain(t *testing.T) []string {
	t.Helper()
	_, chain, _ := strings.Cut(sourcesTable(t), "chain from_guest {\n")
	chain, _, _ = strings.Cut(chain, "\n    }\n")
	var rules []string
	for line := range strings.Lines(chain) {
		rules = append(rules, strings.TrimSpace(line))
	}
	return rules
}

// A source that no guest owns leaves a guest's port toward its own slot alone, as the VPN's reply
// to a peer. A lease is asked from 0.0.0.0. Any other IPv4 source drops, as Incus's NAT would let
// it reach the LAN unchanged.
func TestSourcesTablePassesAForeignSourceTowardTheGuestsSlotAlone(t *testing.T) {
	got := guestChain(t)
	want := []string{
		"ether type ip ip saddr . iifname @owned4 accept",
		"ether type ip ip saddr { 10.100.0.0/24, 10.99.0.0/20 } drop",
		"ether type ip ip saddr 0.0.0.0 ip daddr 255.255.255.255 udp dport 67 accept",
		`iifname "ws-lab" ether type ip ip daddr 10.99.1.0/24 accept`,
		"ether type ip drop",
	}
	if len(got) < len(want) || !slices.Equal(got[len(got)-len(want):], want) {
		t.Errorf("the chain ends with:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// The table's hooks see the frames of every bridge of the host, Docker's too. A frame enters the
// chain from Incus's bridge alone, as the chain's last rule would cut Docker's containers off.
func TestSourcesTableFiltersIncusBridgeAlone(t *testing.T) {
	table := sourcesTable(t)
	scoped := `meta ibrname "` + BridgeName + `" jump from_guest`
	if jumps, ours := strings.Count(table, "jump from_guest"), strings.Count(table, scoped); jumps != 2 || ours != 2 {
		t.Errorf("%d of the %d jumps to from_guest are %q, want both hooks' jumps", ours, jumps, scoped)
	}
}
