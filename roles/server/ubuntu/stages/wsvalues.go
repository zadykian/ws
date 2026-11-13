package stages

import (
	"context"
	"errors"
	"fmt"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"io/fs"
	"strconv"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// valuesStage names the stage in its messages. outboundKey is the key of host_vars that the target
// names, as a cloud VM turns the outbound table off, and userKey the user that ws is set up for.
const valuesStage, outboundKey, userKey = "ws's values", "security_outbound", "ws_user"

// WsValues is the stage of ws's values on ws itself, which ws server ubuntu wizard runs. It asks
// each key that host_vars lacks, and names it there even at its default. So a later run, from ws or
// the Mac, reads the file and asks nothing.
type WsValues struct {
	// Vars is host_vars' path. Set writes one key of it, as ws server ubuntu values set does.
	Vars string
	Set  func(key, value string) error
	// User is the user that ran sudo, which ws_user offers first. Empty at a root login.
	User string
	// Tty tells whether a terminal is there to ask.
	Tty bool
	M   effect.Machine
	P   effect.Prompt
}

// Name is the stage's name.
func (*WsValues) Name() string { return valuesStage }

// Side is ws: its host_vars.
func (*WsValues) Side() guided.Side { return session.Ws }

// WaitsOn is empty: the stage is ws's first.
func (*WsValues) WaitsOn() string { return "" }

// valuesAsk is the change that asks the keys that host_vars lacks, and names them there.
type valuesAsk struct{ vars, keys string }

func (c valuesAsk) String() string {
	return valuesStage + ": would ask " + c.keys + ", and name each in " + c.vars
}

// Plan finds the keys that host_vars lacks.
func (v *WsValues) Plan(context.Context) (step.Plan, error) {
	data, err := v.M.ReadFile(v.Vars)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return step.Plan{}, err
	}
	var lacks []string
	for _, key := range []string{outboundKey, userKey} {
		if !values.Named(data, key) {
			lacks = append(lacks, key)
		}
	}
	if len(lacks) == 0 {
		return step.Plan{}, nil
	}
	return step.Plan{Changes: []step.Change{valuesAsk{v.Vars, strings.Join(lacks, " and ")}}}, nil
}

// Apply asks each key that host_vars lacks, and names it there once answered. Without a terminal it
// stops, naming the command that names a key by hand.
func (v *WsValues) Apply(ctx context.Context, p step.Plan) error {
	c, ok := p.Changes[0].(valuesAsk)
	if !ok {
		return fmt.Errorf("%s: unknown change %T", valuesStage, p.Changes[0])
	}
	if key := strings.Fields(c.keys)[0]; !v.Tty {
		return fmt.Errorf("%s: no terminal to ask %s: name it by sudo ws server ubuntu values set %s VALUE",
			valuesStage, key, key)
	}
	cloud, err := v.target(ctx, c.keys)
	if err != nil || !strings.Contains(c.keys, userKey) {
		return err
	}
	return v.askUser(ctx, cloud)
}

// target tells whether ws is a cloud VM: it asks where keys holds the target, and names the answer
// in host_vars, else reads the target that host_vars names.
func (v *WsValues) target(ctx context.Context, keys string) (bool, error) {
	if !strings.Contains(keys, outboundKey) {
		vals, err := values.Load(v.M, v.Vars)
		return !vals.Security.Outbound, err
	}
	target, err := effect.Choose(ctx, v.P, effect.ChoiceQuestion{
		Text:    "ws's target: lan for a machine on the LAN or a VM on the Mac, cloud for a cloud VM",
		Options: []string{"lan", "cloud"}, Default: "lan"})
	if err != nil {
		return false, err
	}
	return target == "cloud", v.set(outboundKey, strconv.FormatBool(target != "cloud"))
}

// set names key in host_vars, with the stage's name before an error.
func (v *WsValues) set(key, value string) error {
	if err := v.Set(key, value); err != nil {
		return fmt.Errorf("%s: %w", valuesStage, err)
	}
	return nil
}
