package wizard

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

// cableStage names the stage in its messages.
const cableStage = "Cable"

// cableEnd is ws's end of the cable, given to its port by the network role. The Mac's service gets
// the other end.
const cableEnd = "10.77.0.1"

// Cable is the stage of the cable from the Mac's display to a port of ws's, outside the router, on
// a machine on the LAN alone.
//
// Docked, the Mac's cable gives that port carrier, which the files script's link lists among ws's
// wired ports. The stage names the port in /etc/ws/host_vars.yml on ws, whose network role gives it
// 10.77.0.1 as ws's end. Then it names the Mac's service of the cable in mac.conf's
// WS_CABLE_SERVICE, and ws mac apply gives that 10.77.0.2 as the Mac's end. It names the port in
// mac.conf's WS_CABLE_PORT too, which the stage names on ws again after ws's install. Undocked, the
// first setup waits, and the closing list says to dock the Mac.
type Cable struct {
	// Ws is the session to ws, whose files script lists and names the ports. ws is read only where
	// its ssh logs in, in this run.
	Ws *Session
	// SSH is the Mac's ssh to ws.
	SSH SSH
	// Conf is mac.conf's path. Check is set under --check.
	Conf  string
	Check bool
	// RunDone is ws's run's verdict, which the stage waits on under --check.
	RunDone func(ctx context.Context, c conf.Conf) bool
	// MacApply runs ws mac apply, as the Setup stage's Run does, which gives the Mac's service its
	// address.
	MacApply func(ctx context.Context, check bool) (bool, error)
	// Env looks up a variable of the environment.
	Env func(string) (string, bool)
	P   effect.Prompt
	// Prog prefixes the messages of a refused answer.
	Prog string
	Tty  bool
	M    effect.Machine
	Out  effect.Out

	// ports is what link listed on ws at the last read: a line per port.
	ports []cablePort
}

// cablePort is a line of link: a wired port of ws's, its carrier, and its IPv4 addresses. The
// carrier is 1 where a cable plugs in, 0, or - where the port is down.
type cablePort struct{ name, carrier, addrs string }

var cableEndAt = regexp.MustCompile(`(^| )` + regexp.QuoteMeta(cableEnd) + `/`)

// Name is the stage's name.
func (*Cable) Name() string { return cableStage }

// WaitsOn is Homebrew: under --check on a new Mac, Answers is skipped and writes no mac.conf to
// read. Plan says itself that the stage is after ws's run, as wizard-cable.sh's cable_after did.
// That run makes /etc/ws/host_vars.yml on ws, where it has no cable to see off the LAN.
func (*Cable) WaitsOn() string { return homebrewStage }

type cableChange string

func (c cableChange) String() string { return cableStage + ": would " + string(c) }

// read sets the ports to what ws lists. It lists them once port holds 10.77.0.1, or after 10 s, in
// one login, where port is set.
func (c *Cable) read(ctx context.Context, port string) error {
	var out []byte
	var err error
	if port == "" {
		out, err = c.Ws.ports(ctx)
	} else {
		out, err = c.Ws.waitPorts(ctx, port)
	}
	if err != nil {
		return fmt.Errorf("%s: link failed on ws: %w", cableStage, err)
	}
	c.ports = nil
	for _, l := range strings.Split(string(out), "\n") {
		if f := strings.Fields(l); len(f) >= 2 {
			c.ports = append(c.ports, cablePort{f[0], f[1], strings.Join(f[2:], " ")})
		}
	}
	return nil
}

// names returns the ports' names.
func (c *Cable) names() []string {
	var n []string
	for _, p := range c.ports {
		n = append(n, p.name)
	}
	return n
}

// up returns the ports with carrier.
func (c *Cable) up() []string {
	var n []string
	for _, p := range c.ports {
		if p.carrier == "1" {
			n = append(n, p.name)
		}
	}
	return n
}

// end returns the port that holds ws's end of the cable, or an empty string.
func (c *Cable) end() string {
	for _, p := range c.ports {
		if cableEndAt.MatchString(p.name + " " + p.carrier + " " + p.addrs) {
			return p.name
		}
	}
	return ""
}

// lacks tells whether mac.conf has no line of WS_CABLE_SERVICE yet.
func (c *Cable) lacks() bool {
	text, _ := c.M.ReadFile(c.Conf)
	return !regexp.MustCompile(`(?m)^[ \t]*WS_CABLE_SERVICE=`).Match(text)
}

// Plan is empty off the LAN, and where mac.conf's WS_CABLE_SERVICE is empty, or its WS_CABLE_PORT
// is empty or the cable that ws's status names. Without the service, Plan is empty too where no
// port that link lists has carrier, as the Mac isn't docked. The stage waits then, and Hand names
// the dock in the closing list. WS_CABLE_PORT in the environment ends the wait, as for a port that
// ws keeps down. It reads ws only where the login reached it in this run. Under --check it says
// that the stage is after ws's run, where that run has changes to make.
func (c *Cable) Plan(ctx context.Context) (step.Plan, error) {
	cf, err := c.Ws.want()
	if err != nil || cf.Target != "lan" {
		return step.Plan{}, err
	}
	// Under --check, no direct way to ws yet means that ws's run is not done either. in stays
	// false, and the plan says the stage comes after it, as wizard-cable.sh's cable_after did.
	w, err := c.SSH.Way(ctx)
	if err != nil && !c.Check {
		return step.Plan{}, fmt.Errorf("%s: %w", cableStage, err)
	}
	in := false
	if err == nil {
		in, _ = c.SSH.login(ctx, w)
	}
	_, portSet := c.Env("WS_CABLE_PORT")
	svc, svcSet := c.Env("WS_CABLE_SERVICE")
	none := svcSet && svc == ""
	if !c.lacks() && (cf.CableService == "" || cf.CablePort == "" ||
		in && c.holds(ctx, cf.CablePort)) {
		return step.Plan{}, nil
	}
	if c.Check && !none && (!in || !c.RunDone(ctx, cf)) {
		c.Out.Say("%s: after %s", cableStage, runStage)
		return step.Plan{}, nil
	}
	if !c.lacks() {
		return c.change("name " + cf.CablePort + " in /etc/ws/host_vars.yml on ws again, and run " +
			"ws server ubuntu apply --only network,wireguard there")
	}
	if none {
		return c.change("write an empty WS_CABLE_SERVICE in mac.conf, for no cable")
	}
	if !in {
		return c.change("ask for the cable's port and service")
	}
	if portSet && !c.Check {
		// Apply reads the ports once, and names the port that the environment gives.
		return c.change("ask ws's port of the cable, name it in /etc/ws/host_vars.yml on ws, and run " +
			"ws server ubuntu apply --only network,wireguard there")
	}
	if err := c.read(ctx, ""); err != nil {
		return step.Plan{}, err
	}
	if !portSet && len(c.up()) == 0 {
		return step.Plan{}, nil
	}
	if c.end() == "" {
		return c.change("ask ws's port of the cable, name it in /etc/ws/host_vars.yml on ws, and run " +
			"ws server ubuntu apply --only network,wireguard there")
	}
	return c.change("ask the Mac's service of the cable, write it in mac.conf as WS_CABLE_SERVICE, run " +
		"ws client macos apply, which gives it 10.77.0.2, and ping " + cableEnd)
}

func (*Cable) change(text string) (step.Plan, error) {
	return step.Plan{Changes: []step.Change{cableChange(text)}}, nil
}
