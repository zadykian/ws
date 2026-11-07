package wizard

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// valuesStage names the stage in its messages, and outboundKey is the key of host_vars that the
// target names: a cloud VM turns the outbound table off.
const valuesStage, outboundKey = "ws's values", "security_outbound"

// WsValues is the stage of ws's values on ws itself, which ws server wizard runs. It asks each key
// that host_vars lacks, and names it there even at its default, so that a later run, from ws or the
// Mac, reads the file and asks nothing.
type WsValues struct {
	// Vars is host_vars' path. Set writes one key of it, as ws server values set does.
	Vars string
	Set  func(key, value string) error
	// Tty tells whether a terminal is there to ask.
	Tty bool
	M   effect.Machine
	P   effect.Prompt
}

// Name is the stage's name.
func (*WsValues) Name() string { return valuesStage }

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
	for _, key := range []string{outboundKey} {
		if !values.Named(data, key) {
			lacks = append(lacks, key)
		}
	}
	if len(lacks) == 0 {
		return step.Plan{}, nil
	}
	return step.Plan{Changes: []step.Change{valuesAsk{v.Vars, strings.Join(lacks, " and ")}}}, nil
}

// Apply asks the target, and names it in host_vars. Without a terminal it stops, naming the command
// that names a key by hand.
func (v *WsValues) Apply(ctx context.Context, p step.Plan) error {
	c, ok := p.Changes[0].(valuesAsk)
	if !ok {
		return fmt.Errorf("%s: unknown change %T", valuesStage, p.Changes[0])
	}
	if key := strings.Fields(c.keys)[0]; !v.Tty {
		return fmt.Errorf("%s: no terminal to ask %s: name it by sudo ws server values set %s VALUE",
			valuesStage, key, key)
	}
	target, err := effect.Choose(ctx, v.P, effect.ChoiceQuestion{
		Text:    "ws's target: lan for a machine on the LAN or a VM on the Mac, cloud for a cloud VM",
		Options: []string{"lan", "cloud"}, Default: "lan"})
	if err != nil {
		return err
	}
	return v.set(outboundKey, strconv.FormatBool(target != "cloud"))
}

// set names key in host_vars, with the stage's name before an error.
func (v *WsValues) set(key, value string) error {
	if err := v.Set(key, value); err != nil {
		return fmt.Errorf("%s: %w", valuesStage, err)
	}
	return nil
}
