package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/engine/core/effect/units"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/apt"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/server/ubuntu"
	ubuntucli "github.com/zadykian/ws/roles/server/ubuntu/cli"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

const (
	serverVars = "/etc/ws/host_vars.yml"
	osRelease  = "/etc/os-release"
)

// serverRefusal returns why ws server ubuntu apply does not run here, or "" where it does: as root
// on Ubuntu 26.04, the one system that the steps are written for. osRelease is /etc/os-release.
func serverRefusal(euid int, osRelease []byte) string {
	if euid != 0 {
		return "run it as root"
	}
	var id, version string
	sc := bufio.NewScanner(bytes.NewReader(osRelease))
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), "=")
		if !ok {
			continue
		}
		v = strings.Trim(v, `"'`)
		switch k {
		case "ID":
			id = v
		case "VERSION_ID":
			version = v
		}
	}
	if id != "ubuntu" || version != "26.04" {
		return fmt.Sprintf("Ubuntu 26.04 is the one system for this, not %s %s", id, version)
	}
	return ""
}

// newServerCmd returns ws server, the group of the server's systems. It builds everywhere, so that
// the tests run anywhere. Only its entry in the groups is built for Linux alone.
func newServerCmd(p cli.Env) *cobra.Command {
	return cli.Group("server", "the server's systems", newUbuntuCmd(p))
}

// newUbuntuCmd returns ws server ubuntu, the commands that set up an Ubuntu server.
func newUbuntuCmd(p cli.Env) *cobra.Command {
	return cli.Group("ubuntu", "the commands for an Ubuntu server", newApplyCmd(p), newStateCmd(p), newFansCmd(p),
		ubuntucli.NewValuesCmd(p), ubuntucli.NewSecretsCmd(p), newServerWizardCmd(p), newServerConfigureCmd(p))
}

// applyOptions are apply's flags: the runner's, and the --secrets path, which the runner never reads.
type applyOptions struct {
	role.Options
	// Secrets is the age-encrypted file of --secrets, or empty.
	Secrets string
}

// newApplyCmd returns ws server ubuntu apply.
func newApplyCmd(p cli.Env) *cobra.Command {
	var opts applyOptions
	c := &cobra.Command{
		Use:   "apply",
		Short: "set the server up, each role as Go steps",
		Long: "--secrets reads an age-encrypted TOML file with the secrets. The X25519 identity comes from the " +
			"unit's credential ws-secrets (systemd's LoadCredential), in $CREDENTIALS_DIRECTORY.\n\n" +
			"--only names a role, or a step of it as ROLE/STEP, by the name that the run shows. A role's other steps " +
			"are planned and left out, and a step that waits on one of them with changes is skipped.\n\n" +
			"The last line names the roles that changed.\n\n" +
			"--json prints each event of the run as a line of JSON on stdout: planned, started, done, failed " +
			"or skipped, with the step, the first line of each change and the error. ws's own lines stay on " +
			"stderr, and no screen is drawn.",
		Args: cobra.NoArgs,
		RunE: cli.Apply(serverTarget(p, &opts), p),
	}
	f := c.Flags()
	f.Bool("json", false, "print each event of the run as a line of JSON on stdout")
	f.Bool("check", false, "plan the steps and apply none")
	f.StringSliceVar(&opts.Only, "only", nil, "run the named roles and ROLE/STEPs alone, as a list or by repeating the flag")
	f.StringVar(&opts.Secrets, "secrets", "", "read the secrets from the age-encrypted `FILE`")
	return c
}

// serverRun is what a command that reads or sets the server up has made ready.
type serverRun struct {
	cfg    role.Config
	list   []role.Role
	values values.Values
	term   *effect.Terminal
	out    engine.Lines
}

// prepareServer checks where the command runs, then loads the values and the secrets that the
// roles read. The steps of --only follow the values, so they are checked last.
func prepareServer(call engine.Call, p cli.Env, opts applyOptions) (serverRun, error) {
	if call.Selected && len(opts.Only) == 0 {
		return serverRun{}, engine.UsageError{Err: errors.New("--only names no role")}
	}
	_, err := role.Select(ubuntu.Roles(values.Values{}, secrets.Secrets{}), opts.Only)
	if err != nil {
		return serverRun{}, engine.UsageError{Err: err}
	}
	text, err := p.Machine.ReadFile(osRelease)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return serverRun{}, err
	}
	if why := serverRefusal(p.Euid, text); why != "" {
		return serverRun{}, errors.New(why)
	}
	term := effect.NewTerminal(strings.NewReader(""), call.Err)
	out := linesOf(term, call.Path)
	v, err := values.Load(p.Machine, serverVars)
	if err != nil {
		return serverRun{}, err
	}
	// ws_home is known now: the machine's writes under it stay inside it.
	if m, ok := p.Machine.(local.OSMachine); ok {
		m.Confine = v.Home
		p.Machine = m
	}
	sec, err := loadSecrets(p.Machine, opts.Secrets, p.Getenv)
	if err != nil {
		return serverRun{}, err
	}
	cfg := role.Config{Options: opts.Options, M: p.Machine, U: units.OSUnits{}, Out: out, Observer: out,
		Prepare: prepareSteps(p.Machine, out)}
	r := serverRun{cfg: cfg, values: v, term: term, out: out}
	if r.list, err = r.roles(sec); err != nil {
		return serverRun{}, engine.UsageError{Err: err}
	}
	if err := cfg.CheckSteps(r.list); err != nil {
		return serverRun{}, engine.UsageError{Err: err}
	}
	return r, nil
}

// roles returns the roles that --only names, closed over r's values and the secrets s.
func (r serverRun) roles(s secrets.Secrets) ([]role.Role, error) {
	return role.Select(ubuntu.Roles(r.values, s), r.cfg.Only)
}

// prepareSteps is what every role needs: ca-certificates serves the steps' downloads over HTTPS.
func prepareSteps(m effect.Machine, out effect.Out) []step.Step {
	return []step.Step{resource.Group{Title: "Packages", Parts: []resource.Resource{apt.Packages{
		M: m, Out: out, Names: []string{"ca-certificates"},
	}}}}
}

// onScreen is cfg as a work beside a screen takes it: on the screen's effects, with its output and
// its events on b.
func onScreen(cfg role.Config, b *tui.Bridge) role.Config {
	e, o := b.Effects(), b.Options(step.Options{})
	cfg.M, cfg.Observer, cfg.Stop = e.M, o.Observer, o.Stop
	cfg.Out, cfg.Prepare = b, prepareSteps(e.M, b)
	return cfg
}

// loadSecrets reads the identity from the unit's credential and decrypts the file, or returns no
// secrets where path is empty.
func loadSecrets(m effect.Machine, path string, getenv func(string) string) (secrets.Secrets, error) {
	if path == "" {
		return secrets.Secrets{}, nil
	}
	id, err := secrets.ReadCredential(m, getenv)
	if err != nil {
		return secrets.Secrets{}, err
	}
	s, err := secrets.Load(m, path, id)
	s.Whole = err == nil
	return s, err
}
