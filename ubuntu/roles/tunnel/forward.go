package tunnel

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/ubuntu/values"
)

// Forward sends all that arrives on a guest's interface to that guest, whatever its destination. A
// rule on the interface selects a table of the slot, whose one route leads to the guest's address
// on Bridge. No NAT, so the guest sees the peer's tunnel address. Without the rule, as at boot or
// after a flush, the main table would route a peer's packets to any guest. A rule or a route of a
// slot's priority range that no tunnel wants is deleted, one of the right priority but another
// interface or table among them.
type Forward struct {
	M       effect.Machine
	Tunnels []values.GuestTunnel
	// Bridge is the interface that the guests' addresses are on.
	Bridge string
}

// priorities are the rules that ws owns: those of each slot's priority.
const (
	firstPriority = 1000
	lastPriority  = firstPriority + 15
)

type ipRule struct {
	Priority int    `json:"priority"`
	IIF      string `json:"iif"`
	Table    string `json:"table"`
}

// ipRoute is a route of `ip -j -4 route show table all`, which names the table of each route but
// main's.
type ipRoute struct {
	Dst     string `json:"dst"`
	Gateway string `json:"gateway"`
	Dev     string `json:"dev"`
	Table   string `json:"table"`
}

// readJSON runs ip with -j -4 and decodes its answer.
func (f Forward) readJSON(ctx context.Context, into any, args ...string) error {
	out, err := f.M.Run(ctx, "ip", append([]string{"-j", "-4"}, args...)...)
	if err != nil {
		return fmt.Errorf("ip %s: %w", strings.Join(args, " "), err)
	}
	if err := json.Unmarshal(out, into); err != nil {
		return fmt.Errorf("ip %s: its output is no JSON of its entries: %w", strings.Join(args, " "), err)
	}
	return nil
}

// Plan plans the rule and the route of each tunnel that lack, and the deletion of a rule of a
// slot's priority that no tunnel wants.
func (f Forward) Plan(ctx context.Context) ([]resource.Change, error) {
	var rules []ipRule
	if err := f.readJSON(ctx, &rules, "rule", "show"); err != nil {
		return nil, err
	}
	// All tables at once: ip exits 2 on a table that holds no route, as before the first run and
	// after a boot. A read of that table alone would stop the plan.
	var routes []ipRoute
	if err := f.readJSON(ctx, &routes, "route", "show", "table", "all"); err != nil {
		return nil, err
	}
	var changes []resource.Change
	for _, t := range f.Tunnels {
		table := strconv.Itoa(t.Table())
		if !hasRoute(routes, table, t.Guest.Address, f.Bridge) {
			changes = append(changes, addRoute{f.M, t, f.Bridge})
		}
		if !hasRule(rules, t) {
			changes = append(changes, addRule{f.M, t})
		}
	}
	for _, r := range rules {
		if r.Priority >= firstPriority && r.Priority <= lastPriority && strings.HasPrefix(r.IIF, "wg") && !f.wants(r) {
			changes = append(changes, deleteRule{f.M, r})
		}
	}
	return changes, nil
}

func hasRoute(routes []ipRoute, table, gateway, dev string) bool {
	for _, r := range routes {
		if r.Table == table && r.Dst == "default" && r.Gateway == gateway && r.Dev == dev {
			return true
		}
	}
	return false
}

// isRuleOf is whether r is the rule that sends t's interface to t's table, at t's priority.
func isRuleOf(r ipRule, t values.GuestTunnel) bool {
	return r.Priority == t.Table() && r.IIF == t.Link() && r.Table == strconv.Itoa(t.Table())
}

func hasRule(rules []ipRule, t values.GuestTunnel) bool {
	return slices.ContainsFunc(rules, func(r ipRule) bool { return isRuleOf(r, t) })
}

// wants is whether a tunnel wants the rule r. A rule of a slot's priority with another interface or
// table is no tunnel's, so it goes and the right one comes.
func (f Forward) wants(r ipRule) bool {
	return slices.ContainsFunc(f.Tunnels, func(t values.GuestTunnel) bool { return isRuleOf(r, t) })
}

type addRoute struct {
	m      effect.Machine
	t      values.GuestTunnel
	bridge string
}

func (c addRoute) String() string {
	return fmt.Sprintf("route table %d to %s's address %s on %s", c.t.Table(), c.t.Guest.Name, c.t.Guest.Address, c.bridge)
}

func (c addRoute) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, "ip", "-4", "route", "replace", "default", "via", c.t.Guest.Address, "dev", c.bridge, "onlink", "table", strconv.Itoa(c.t.Table()))
	return err
}

type addRule struct {
	m effect.Machine
	t values.GuestTunnel
}

func (c addRule) String() string {
	return fmt.Sprintf("send what arrives on %s to table %d, %s's", c.t.Link(), c.t.Table(), c.t.Guest.Name)
}

func (c addRule) Apply(ctx context.Context) error {
	table := strconv.Itoa(c.t.Table())
	_, err := c.m.RunRoot(ctx, "ip", "-4", "rule", "add", "iif", c.t.Link(), "lookup", table, "priority", table)
	return err
}

type deleteRule struct {
	m effect.Machine
	r ipRule
}

func (c deleteRule) String() string {
	return fmt.Sprintf("delete the rule of priority %d for %s, which no guest's tunnel wants", c.r.Priority, c.r.IIF)
}

func (c deleteRule) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, "ip", "-4", "rule", "delete", "iif", c.r.IIF, "priority", strconv.Itoa(c.r.Priority), "lookup", c.r.Table)
	return err
}
