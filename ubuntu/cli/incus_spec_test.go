package cli

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/ufw"
	"github.com/zadykian/ws/ubuntu/values"
	"io"

	"errors"
	enginecli "github.com/zadykian/ws/engine/cli"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/ubuntu/roles/incus"
)

const incusPoolCommand = asRoot + selfPath + " ubuntu host incus pool"

// incusUnits loads the host's units for the default values.
func incusUnits(t *testing.T, m *effecttest.Machine) []kind.Unit {
	units, _ := cableUnits(t, m, "---\n")
	return units
}

func TestIncusPoolRunsWhereItsCheckFails(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[incusPoolCommand+" --check"] = effecttest.Result{Err: errors.New("to do: create the storage pool ws")}
	m.Results[incusPoolCommand] = effecttest.Result{}
	got := applyUnit(t, incusUnits(t, m), "state/incus-pool")
	if len(got) != 1 || !strings.Contains(got[0], selfPath+" ubuntu host incus pool") {
		t.Errorf("plan %q, want the pool's command", got)
	}
	if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasSuffix(c, incusPoolCommand) && !strings.HasSuffix(c, "--check") }) {
		t.Errorf("calls %q, want the command run as root", m.Calls)
	}
}

func TestIncusPoolIsInSyncWhereItsCheckPasses(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[incusPoolCommand+" --check"] = effecttest.Result{}
	if got := applyUnit(t, incusUnits(t, m), "state/incus-pool"); got != nil {
		t.Errorf("plan %q, want none", got)
	}
}

func TestIncusUnitsRunTheObjectsThenTheSetThenTheGuests(t *testing.T) {
	units := incusUnits(t, effecttest.NewMachine())
	var order []string
	for _, u := range units {
		if strings.HasPrefix(u.Name, "state/incus-") && u.Name != "state/incus-ufw" {
			order = append(order, u.Name)
		}
	}
	want := []string{"state/incus-acl", "state/incus-ingress", "state/incus-pool", "state/incus-bridge", "state/incus-profile", "state/incus-image", "state/incus-owned", "state/incus-guests"}
	if !slices.Equal(order, want) {
		t.Errorf("units %q, want %q", order, want)
	}
}

func TestEachIncusUnitRunsItsOwnCommand(t *testing.T) {
	for _, object := range []string{"acl", "ingress", "bridge", "profile", "image", "owned", "guests"} {
		command := asRoot + selfPath + " ubuntu host incus " + object
		m := effecttest.NewMachine()
		m.Results[command+" --check"] = effecttest.Result{Err: errors.New("to do")}
		m.Results[command] = effecttest.Result{}
		if got := applyUnit(t, incusUnits(t, m), "state/incus-"+object); len(got) != 1 {
			t.Errorf("%s: plan %q, want the command", object, got)
		}
	}
}

// runIncus runs ws ubuntu host incus ARGS on m, with the host's default values.
func runIncus(m *effecttest.Machine, args ...string) error {
	c := newIncusCmd(enginecli.Env{Machine: m})
	c.SetArgs(args)
	c.SetOut(io.Discard)
	c.SetErr(io.Discard)
	return c.Execute()
}

func TestIncusCommandsCheckAndMakeTheirObject(t *testing.T) {
	for _, tc := range []struct{ object, list, create string }{
		{"acl", "network acl list", "network acl create ws-egress"},
		{"ingress", "network acl list", "network acl create ws-ingress"},
		{"bridge", "network list", "network create incusbr0 --type bridge"},
		{"profile", "profile list", "profile create ws"},
	} {
		m := effecttest.NewMachine()
		m.Results["incus "+tc.list+" --format json"] = effecttest.Result{Out: []byte("[]")}
		m.Results["incus "+tc.create] = effecttest.Result{}
		if err := runIncus(m, tc.object, "--check"); !errors.Is(err, enginecli.ErrDue) {
			t.Errorf("%s --check = %v, want a change due", tc.object, err)
		}
		if len(m.Inputs) != 0 {
			t.Errorf("%s --check wrote %q", tc.object, m.Inputs)
		}
		if err := runIncus(m, tc.object); err != nil || len(m.Inputs) != 1 {
			t.Errorf("%s = %v, with %d writes, want one create", tc.object, err, len(m.Inputs))
		}
	}
}

func TestIncusImageChecksTheAliasOfItsMetadata(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results["incus image list --format json"] = effecttest.Result{Out: []byte("[]")}
	if err := runIncus(m, "image", "--check"); !errors.Is(err, enginecli.ErrDue) {
		t.Errorf("image --check with no image = %v, want a change due", err)
	}
	alias, err := incus.Image{M: m}.Alias(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	m.Results["incus image list --format json"] = effecttest.Result{Out: []byte(`[{"aliases":[{"name":"` + alias + `"}]}]`)}
	if err := runIncus(m, "image", "--check"); err != nil {
		t.Errorf("image --check with the image = %v, want none due", err)
	}
}

func TestIncusPoolWithNoPartitionFailsNamingTheKey(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["incus storage list --format json"] = effecttest.Result{Out: []byte("[]")}
	if err := runIncus(m, "pool"); err == nil || !strings.Contains(err.Error(), "WS_INCUS_POOL_SOURCE") {
		t.Errorf("pool = %v, want the key named", err)
	}
}

func TestIncusUFWLetsDHCPInAndRoutesTheBridge(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[ufw.UserRules] = nil
	want := []string{
		"ufw allow in on incusbr0 from any to any port 67 proto udp",
		"ufw route allow in on incusbr0 from any to any",
	}
	for _, w := range want {
		m.Results[w+" comment incus: DHCP and routes from the guests"] = effecttest.Result{}
	}
	got := applyUnit(t, incusUnits(t, m), "state/incus-ufw")
	if len(got) != len(want) {
		t.Fatalf("plan %q, want the two rules", got)
	}
	for _, w := range want {
		if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, w) }) {
			t.Errorf("calls %q, want %q", m.Calls, w)
		}
	}
	for _, c := range m.Calls {
		if strings.Contains(c, "53") || strings.Contains(c, "out on") {
			t.Errorf("call %q: DNS stays shut, and no rule names an out interface", c)
		}
	}
}

const (
	sourcesFile = "/usr/local/lib/ws/incus/sources.nft"
	// sourcesInForce and sourcesLoads are the commands of service/incus-sources's drift, as the
	// fake keys them.
	sourcesInForce = "sh -c nft -s -t list table bridge ws_sources && nft -s -t list table ip ws_guest_out"
	sourcesLoads   = `unshare --net sh -c nft -f "$1" && nft -s -t list table bridge ws_sources && nft -s -t list table ip ws_guest_out sh ` + sourcesFile
)

// sourcesMachine is a machine whose table checks, with the table in force as in.
func sourcesMachine(in string) (*effecttest.Machine, *effecttest.Units) {
	m := effecttest.NewMachine()
	m.Results["/usr/sbin/nft -c -f "+sourcesFile+".ws-check"] = effecttest.Result{}
	m.Results[sourcesInForce] = effecttest.Result{Out: []byte(in)}
	m.Results[sourcesLoads] = effecttest.Result{Out: []byte("chain\n")}
	return m, &effecttest.Units{States: map[string]effect.UnitState{"incus-sources.service": {Active: true}}}
}

func TestSourcesTableDropsWhatAGuestMayNotClaim(t *testing.T) {
	m, _ := sourcesMachine("chain\n")
	applyUnit(t, incusUnits(t, m), "service/incus-sources")
	table := string(m.Files[sourcesFile])
	ranges := "{ 10.100.0.0/24, 10.99.0.0/20 }"
	for _, rule := range []string{
		"ether type arp arp saddr ip . iifname @owned4 accept",
		"ether type arp arp saddr ip " + ranges + " drop",
		"ether type ip ip saddr . iifname @owned4 accept",
		"ether type ip ip saddr " + ranges + " drop",
		"type filter hook input priority -300; policy accept;",
		"type filter hook forward priority -300; policy accept;",
	} {
		if !strings.Contains(table, rule) {
			t.Errorf("the table lacks %q:\n%s", rule, table)
		}
	}
	// A guest's own address passes before the drop, and the rest of the traffic is no rule's.
	for _, family := range []string{"arp saddr ip", "ip saddr"} {
		own, drop := strings.Index(table, family+" . iifname @owned4 accept"), strings.Index(table, family+" { ")
		if own < 0 || drop < own {
			t.Errorf("the %s drop comes before its owner's accept:\n%s", family, table)
		}
	}
	for _, line := range strings.Split(table, "\n") {
		if code, _, _ := strings.Cut(line, "#"); strings.Contains(code, "br_netfilter") || strings.Contains(code, "ipv4_filtering") {
			t.Errorf("the table leans on br_netfilter or Incus's IPv4 filter: %s", line)
		}
	}
}

// A VLAN tag, or any ether type that no rule reads, would hide a source from the rules, which the
// host would strip and take.
func TestSourcesTableDropsOtherEtherTypesBeforeAnyAccept(t *testing.T) {
	m, _ := sourcesMachine("chain\n")
	applyUnit(t, incusUnits(t, m), "service/incus-sources")
	table := string(m.Files[sourcesFile])
	drop, first := strings.Index(table, "ether type != { arp, ip, ip6 } drop"), strings.Index(table, "@owned4 accept")
	if drop < 0 || first < 0 || drop > first {
		t.Errorf("the drop of other ether types is not the chain's first rule:\n%s", table)
	}
}

func TestSourcesServiceIsRequiredByIncus(t *testing.T) {
	m, _ := sourcesMachine("chain\n")
	applyUnit(t, incusUnits(t, m), "service/incus-sources")
	unit := string(m.Files["/etc/systemd/system/incus-sources.service"])
	if !strings.Contains(unit, "\nRequiredBy=incus.service\n") || !strings.Contains(unit, "\nBefore=") {
		t.Errorf("the unit:\n%s\nwant Incus to require it, so a failed load keeps the guests down", unit)
	}
}

func TestSourcesTableNamesTheRangesOfTheValues(t *testing.T) {
	m, _ := sourcesMachine("chain\n")
	units, _ := cableUnits(t, m, "---\nWS_INCUS_BRIDGE_ADDRESS: 10.100.8.1/22\n")
	applyUnit(t, units, "service/incus-sources")
	if table := string(m.Files[sourcesFile]); !strings.Contains(table, "{ 10.100.8.0/22, 10.99.0.0/20 } drop") {
		t.Errorf("the table lacks the bridge's subnet:\n%s", table)
	}
}

func TestSourcesTableIsCheckedByNftBeforeItGoesLive(t *testing.T) {
	m, _ := sourcesMachine("chain\n")
	applyUnit(t, incusUnits(t, m), "service/incus-sources")
	if !slices.Contains(m.Calls, "root /usr/sbin/nft -c -f "+sourcesFile+".ws-check") {
		t.Errorf("calls %q, want the new file checked by nft", m.Calls)
	}
}

func TestSourcesServiceReloadsWhereTheTableInForceDiffers(t *testing.T) {
	for _, tc := range []struct {
		in      string
		reloads bool
	}{{"chain\n", false}, {"chain flushed by hand\n", true}} {
		m, u := sourcesMachine(tc.in)
		// The file and the unit are in place, so that only the drift is left to plan.
		applyUnit(t, incusUnits(t, m), "service/incus-sources")
		v, err := values.ParseHost(nil)
		if err != nil {
			t.Fatal(err)
		}
		units := loadSpecs(t, kind.Env{M: m, U: u, Out: &effecttest.Out{}, Data: v, Side: "host", Self: selfPath})
		applyUnit(t, units, "service/incus-sources")
		got := slices.ContainsFunc(u.Calls, func(c string) bool {
			return strings.Contains(c, "incus-sources.service") && c != "enable incus-sources.service"
		})
		if got != tc.reloads {
			t.Errorf("in force %q: unit calls %q, want a reload or restart %t", tc.in, u.Calls, tc.reloads)
		}
	}
}

func TestIncusIngressAllowsEachGuestsPeers(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[hostVars] = []byte("WS_INCUS_GUESTS:\n  - {name: lab, address: 10.100.0.10, cpu: 2, memory: 4GiB, disk: 20GiB}\n")
	m.Files["/etc/ws/guests/lab/values.yml"] = []byte("WS_TUNNEL_ADDRESS: 10.99.1.1/24\nWS_TUNNEL_PORT: 51821\n")
	m.Results["incus network acl list --format json"] = effecttest.Result{Out: []byte("[]")}
	m.Results["incus network acl create ws-ingress"] = effecttest.Result{}
	if err := runIncus(m, "ingress"); err != nil || len(m.Inputs) != 1 {
		t.Fatalf("ingress = %v, with %d writes", err, len(m.Inputs))
	}
	if !strings.Contains(m.Inputs[0], "source: 10.99.1.2") || !strings.Contains(m.Inputs[0], "destination: 10.100.0.10,10.99.1.1") {
		t.Errorf("the ACL:\n%s", m.Inputs[0])
	}
}

func TestIncusIngressRefusesAGuestOfTheHostsOwnSlot(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[hostVars] = []byte("WS_INCUS_GUESTS:\n  - {name: lab, address: 10.100.0.10, cpu: 2, memory: 4GiB, disk: 20GiB}\n")
	m.Results["incus network acl list --format json"] = effecttest.Result{Out: []byte("[]")}
	if err := runIncus(m, "ingress"); err == nil || !strings.Contains(err.Error(), "slot 0") {
		t.Errorf("ingress = %v, want a refusal that names slot 0", err)
	}
}
