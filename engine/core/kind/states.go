package kind

import (
	"cmp"
	"errors"
	"fmt"
	"net/netip"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/route"
	"github.com/zadykian/ws/engine/core/resource/ufw"
)

// ufwRule is one rule of a ufw state, in ufw.Rule's fields. An address or a port left out is any.
// With Out, the rule routes (ufw.Route): what comes in on Interface from From may go out by Out.
type ufwRule struct {
	Action    string `toml:"action"`
	Proto     string `toml:"proto"`
	To        string `toml:"to"`
	ToPort    string `toml:"to_port"`
	From      string `toml:"from"`
	FromPort  string `toml:"from_port"`
	Dir       string `toml:"dir"`
	Interface string `toml:"interface"`
	Out       string `toml:"out"`
}

// rule returns the rule of ufw's that r gives, with the state's comment.
func (r ufwRule) rule(comment string) (ufw.Rule, error) {
	if r.Out == "" {
		return ufw.Rule{Action: r.Action, Proto: r.Proto, To: cmp.Or(r.To, "any"), ToPort: cmp.Or(r.ToPort, "any"),
			From: cmp.Or(r.From, "any"), FromPort: cmp.Or(r.FromPort, "any"), Dir: r.Dir, Interface: r.Interface,
			Comment: comment}, nil
	}
	switch {
	case r.Interface == "":
		return ufw.Rule{}, errors.New("a route rule takes interface, where its packets come in")
	case r.Proto != "" || r.To != "" || r.ToPort != "" || r.FromPort != "" || r.Dir != "":
		// ufw.Route fixes these, and Parse reads its rules back in that form alone.
		return ufw.Rule{}, errors.New("a route rule takes action, interface, out and from alone")
	}
	return ufw.Route(r.Action, r.Interface, r.Out, cmp.Or(r.From, "any"), comment), nil
}

// ufwState is the ufw rules that carry Comment, which the state owns: those in Rules are added,
// and the others deleted. Owns narrows the rules it owns by their fields (ufwowns.go). Without
// booted systemd, where ufw cannot load them, it skips.
type ufwState struct {
	Comment string            `toml:"comment"`
	Owns    map[string]string `toml:"owns"`
	Rules   []ufwRule         `toml:"rules"`
}

func (s ufwState) resource(e Env) (resource.Resource, error) {
	if s.Comment == "" {
		return nil, errors.New("a ufw state takes a comment, which marks the rules it owns")
	}
	rules := make([]ufw.Rule, 0, len(s.Rules))
	for i, r := range s.Rules {
		rule, err := r.rule(s.Comment)
		if err == nil {
			err = rule.Valid()
		}
		if err != nil {
			return nil, fmt.Errorf("rule %d: %w", i+1, err)
		}
		rules = append(rules, rule)
	}
	owns, err := s.owns()
	if err != nil {
		return nil, err
	}
	return ufw.Rules{M: e.M, Out: e.Out, Comment: s.Comment, Want: rules, Owns: owns, NeedsSystemd: true}, nil
}

// routesState refuses a route that Pool or Net would overlap, before a program takes them
// (route.Clear). Net is a network that Dev alone routes, and Hint says how to clear an overlap.
type routesState struct {
	Pool string `toml:"pool"`
	Net  string `toml:"net"`
	Dev  string `toml:"dev"`
	Hint string `toml:"hint"`
}

func (s routesState) resource(e Env) (resource.Resource, error) {
	pool, err := netip.ParsePrefix(s.Pool)
	if err != nil {
		return nil, fmt.Errorf("pool: %w", err)
	}
	own, err := netip.ParsePrefix(s.Net)
	if err != nil {
		return nil, fmt.Errorf("net: %w", err)
	}
	if s.Dev == "" {
		return nil, errors.New("a routes state takes dev, the device that routes net")
	}
	return route.Clear{M: e.M, Pool: pool, Net: own, Dev: s.Dev, Hint: s.Hint}, nil
}

// addrState refuses an interface that lacks an address, which the run cannot move (route.Addr).
// Without booted systemd it skips.
type addrState struct {
	Iface string `toml:"iface"`
	Want  string `toml:"want"`
	Hint  string `toml:"hint"`
}

func (s addrState) resource(e Env) (resource.Resource, error) {
	want, err := netip.ParsePrefix(s.Want)
	if err != nil {
		return nil, fmt.Errorf("want: %w", err)
	}
	if s.Iface == "" {
		return nil, errors.New("an addr state takes iface")
	}
	return route.Addr{M: e.M, Out: e.Out, Iface: s.Iface, Want: want, Hint: s.Hint}, nil
}

// runState runs Command as User where something is to do: where a path of Creates is missing, or
// where Check, run as User too, exits other than 0. It takes exactly one of the two. A procedure
// that stays a command of ws runs through it, with that command's own check. On Linux User is root
// alone, and without booted systemd, which root's session needs, it skips, unless Unbooted says
// that the command needs no session. macOS has no systemd, and runs it. There Check runs as ws's
// own user, whatever User, so that a read asks no password (run.go).
type runState struct {
	User    string   `toml:"user"`
	Command []string `toml:"command"`
	Creates []string `toml:"creates"`
	Check   []string `toml:"check"`
	// Unbooted runs the command without booted systemd too, as in CI's container, with no session.
	Unbooted bool `toml:"unbooted"`
	// Path holds directories that the command and its check find before the system's PATH. An
	// installer that puts its tool in one warns where PATH lacks it.
	Path []string `toml:"path"`
}

func (s runState) resource(e Env) (resource.Resource, error) {
	if err := checkUser("user", s.User); err != nil {
		return nil, err
	}
	if e.System == Linux && s.User != "root" {
		return nil, fmt.Errorf("user %q: ws runs a command as root alone on Linux", s.User)
	}
	if len(s.Command) == 0 || s.Command[0] == "" {
		return nil, errors.New("a run state takes a command")
	}
	if (len(s.Creates) == 0) == (len(s.Check) == 0) {
		return nil, errors.New("a run state takes exactly one of creates and check")
	}
	if len(s.Check) > 0 && s.Check[0] == "" {
		return nil, errors.New("check names no command")
	}
	for _, p := range s.Creates {
		if !path.IsAbs(p) || path.Clean(p) != p {
			return nil, fmt.Errorf("creates: %q is not a clean absolute path", p)
		}
	}
	for _, p := range s.Path {
		if !path.IsAbs(p) || path.Clean(p) != p || strings.Contains(p, ":") {
			return nil, fmt.Errorf("path: %q is not a clean absolute path free of ':'", p)
		}
	}
	return runAs{m: e.M, out: e.Out, sys: e.System, self: e.User, ws: e.Self, hidden: e.Hidden, s: s}, nil
}

// runAs is a run state's resource.
type runAs struct {
	m   effect.Machine
	out effect.Out
	sys System
	// self is the account that ws runs as on the Mac.
	self string
	// ws is the path of the ws that runs the units, whose commands keep the hidden names.
	ws     string
	hidden []string
	s      runState
}
