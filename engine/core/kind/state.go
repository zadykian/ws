package kind

import (
	"context"
	"fmt"
	"io/fs"
	"reflect"
	"regexp"
	"strings"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/apt"
	"github.com/zadykian/ws/engine/core/resource/systemd"
	"github.com/zadykian/ws/engine/core/step"
)

// userName is a user's or a group's name as a state takes it: no option for the command it goes
// to, and no path.
var userName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]*\$?$`)

// checkUser refuses a name that is no user's, field naming it in the error.
func checkUser(field, name string) error {
	if !userName.MatchString(name) {
		return fmt.Errorf("%s %q is no user's name", field, name)
	}
	return nil
}

// state is one field of a state unit: a small state with a cheap check.
type state interface {
	// resource checks the state's fields and returns its resource.
	resource(e Env) (resource.Resource, error)
}

// stateSpec is a unit of the kind state, which sets exactly one of its fields. A new state is a
// field here, of a type that implements state, tagged with its key.
type stateSpec struct {
	base
	// Linger is a user whose systemd starts at boot.
	Linger lingerState  `toml:"linger"`
	Purged purgedState  `toml:"purged"`
	HTTP   httpState    `toml:"http"`
	UFW    *ufwState    `toml:"ufw"`
	Routes *routesState `toml:"routes"`
	Addr   *addrState   `toml:"addr"`
	Left   *leftState   `toml:"left"`
	Run    *runState    `toml:"run"`
	Reboot *rebootState `toml:"reboot"`
	// Locale, Timezone and Swap are the system's, in systemstates.go.
	Locale   *localeState   `toml:"locale"`
	Timezone *timezoneState `toml:"timezone"`
	Swap     *swapState     `toml:"swap"`
}

// stateKeys are the keys of stateSpec's states, in their order.
func stateKeys() []string {
	var keys []string
	for f := range reflect.TypeFor[stateSpec]().Fields() {
		if !f.Anonymous {
			keys = append(keys, f.Tag.Get("toml"))
		}
	}
	return keys
}

func (s stateSpec) parts(e Env, _ fs.FS) (func(context.Context) ([]resource.Resource, error), error) {
	var set []string
	var one state
	v := reflect.ValueOf(s)
	for i := range v.NumField() {
		f := v.Type().Field(i)
		if f.Anonymous || v.Field(i).IsZero() {
			continue
		}
		set = append(set, f.Tag.Get("toml"))
		one = v.Field(i).Interface().(state)
	}
	if len(set) != 1 {
		return nil, fmt.Errorf("a state sets exactly one of %s, and this one sets %d", strings.Join(stateKeys(), ", "), len(set))
	}
	// The other states read and change what Linux alone has: systemd, apt, ufw and iproute2.
	if e.System == MacOS && set[0] != "run" {
		return nil, fmt.Errorf("%s: on macOS a state takes run alone", set[0])
	}
	r, err := one.resource(e)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", set[0], err)
	}
	return func(context.Context) ([]resource.Resource, error) { return []resource.Resource{r}, nil }, nil
}

// lingerState is a user whose systemd starts at boot, and runs: user@UID.service. Without booted
// systemd it skips.
type lingerState string

func (s lingerState) resource(e Env) (resource.Resource, error) {
	if err := checkUser("linger", string(s)); err != nil {
		return nil, err
	}
	return lingered{e: e, user: string(s)}, nil
}

// lingered is a linger state's resource. The user's uid, which names the manager's unit, is looked
// up as it plans, as an earlier unit may make the account. The manager's start keeps it running
// where it stopped after lingering began; on a first run loginctl has started it already.
type lingered struct {
	e    Env
	user string
}

// manager is the unit of the user's own systemd.
func (l lingered) manager(ctx context.Context) (systemd.Unit, error) {
	a, err := account.User{Name: l.user}.Lookup(ctx, l.e.M)
	if err != nil {
		return systemd.Unit{}, err
	}
	return systemd.Unit{M: l.e.M, U: l.e.U, Out: l.e.Out, Name: fmt.Sprintf("user@%d.service", a.UID), Run: systemd.Running}, nil
}

func (l lingered) Plan(ctx context.Context) ([]resource.Change, error) {
	linger := systemd.Linger{M: l.e.M, User: account.User{Name: l.user}, Out: l.e.Out}
	cs, err := linger.Plan(ctx)
	if err != nil || !systemd.Booted(l.e.M) {
		return cs, err
	}
	u, err := l.manager(ctx)
	if err != nil {
		return nil, err
	}
	more, err := u.Plan(ctx)
	return append(cs, more...), err
}

// Report reads the manager's state. Plan says that systemd is not running, so this doesn't.
func (l lingered) Report(ctx context.Context) ([]step.Item, error) {
	if !systemd.Booted(l.e.M) {
		return nil, nil
	}
	u, err := l.manager(ctx)
	if err != nil {
		return nil, err
	}
	return u.Report(ctx)
}

// purgedState is a package removed with its settings, where dpkg holds anything of it.
type purgedState string

func (s purgedState) resource(e Env) (resource.Resource, error) {
	if strings.HasPrefix(string(s), "-") || strings.ContainsAny(string(s), " \t/") {
		return nil, fmt.Errorf("%q is no package's name", string(s))
	}
	return apt.Purged{M: e.M, Names: []string{string(s)}}, nil
}
