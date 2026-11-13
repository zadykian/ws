package cmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/engine/core/effect/units"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/apt"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/server/ubuntu"
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
func newServerCmd(p process) *cobra.Command {
	return group("server", "the server's systems", newUbuntuCmd(p))
}

// newUbuntuCmd returns ws server ubuntu, the commands that set up an Ubuntu server.
func newUbuntuCmd(p process) *cobra.Command {
	return group("ubuntu", "the commands for an Ubuntu server", newApplyCmd(p), newStateCmd(p), newFansCmd(p),
		newValuesCmd(p), newServerSecretsCmd(p), newServerWizardCmd(p), newServerConfigureCmd(p))
}

// applyOptions are apply's flags: the runner's, and the --secrets path, which the runner never reads.
type applyOptions struct {
	role.Options
	// Secrets is the age-encrypted file of --secrets, or empty.
	Secrets string
}

// newApplyCmd returns ws server ubuntu apply.
func newApplyCmd(p process) *cobra.Command {
	var opts applyOptions
	var asJSON bool
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
		RunE: func(c *cobra.Command, _ []string) error { return runApply(c, p, opts, asJSON) },
	}
	f := c.Flags()
	f.BoolVar(&asJSON, "json", false, "print each event of the run as a line of JSON on stdout")
	f.BoolVar(&opts.Check, "check", false, "plan the steps and apply none")
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
	out    progOut
}

// prepareServer checks where the command runs, then loads the values and the secrets that the
// roles read. The steps of --only follow the values, so they are checked last.
func prepareServer(c *cobra.Command, p process, opts applyOptions) (serverRun, error) {
	if c.Flags().Changed("only") && len(opts.Only) == 0 {
		return serverRun{}, usageError{errors.New("--only names no role")}
	}
	_, err := role.Select(ubuntu.Roles(values.Values{}, secrets.Secrets{}), opts.Only)
	if err != nil {
		return serverRun{}, usageError{err}
	}
	text, err := p.machine.ReadFile(osRelease)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return serverRun{}, err
	}
	if why := serverRefusal(p.euid, text); why != "" {
		return serverRun{}, errors.New(why)
	}
	term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
	out := progOut{term, c.CommandPath(), map[string]bool{}}
	v, err := values.Load(p.machine, serverVars)
	if err != nil {
		return serverRun{}, err
	}
	// ws_home is known now: the machine's writes under it stay inside it.
	if m, ok := p.machine.(local.OSMachine); ok {
		m.Confine = v.Home
		p.machine = m
	}
	sec, err := loadSecrets(p.machine, opts.Secrets, p.getenv)
	if err != nil {
		return serverRun{}, err
	}
	cfg := role.Config{Options: opts.Options, M: p.machine, U: units.OSUnits{}, Out: out, Observer: out,
		Prepare: prepareSteps(p.machine, out)}
	r := serverRun{cfg: cfg, values: v, term: term, out: out}
	if r.list, err = r.roles(sec); err != nil {
		return serverRun{}, usageError{err}
	}
	if err := cfg.CheckSteps(r.list); err != nil {
		return serverRun{}, usageError{err}
	}
	return r, nil
}

// roles returns the roles that --only names, closed over r's values and the secrets s.
func (r serverRun) roles(s secrets.Secrets) ([]role.Role, error) {
	return role.Select(ubuntu.Roles(r.values, s), r.cfg.Only)
}

// runApply sets the server up as opts say.
func runApply(c *cobra.Command, p process, opts applyOptions, asJSON bool) error {
	r, err := prepareServer(c, p, opts)
	if err != nil {
		return err
	}
	if opensList(c, p, c.Flags().Changed("only")) {
		if r.cfg, r.list, err = pickServer(c, p, r.cfg, r.list); err != nil || r.list == nil {
			r.term.Flush()
			return err
		}
	}
	var events *jsonEvents
	if asJSON {
		events = newJSONEvents(c.OutOrStdout(), r.out)
		r.cfg.Observer = events
	}
	var changed []string
	if drawsScreen(c, p) {
		changed, err = applyOnScreen(c, p, r.cfg, r.list)
	} else {
		changed, err = role.Run(c.Context(), r.cfg, r.list)
	}
	r.term.Flush()
	if err != nil {
		return err
	}
	if events != nil && events.err != nil {
		return events.err
	}
	r.out.Say("%s", role.Summary(changed, r.cfg.Check))
	return nil
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

// applyOnScreen is role.Run under the run screen. cfg's steps name the screen's rows, and the run
// builds them again on the screen's effects. The roles it returns are those of a run that ended.
func applyOnScreen(c *cobra.Command, p process, cfg role.Config, list []role.Role) ([]string, error) {
	var changed []string
	p.machine = cfg.M // ws_home confines it.
	err := drawRun(c, p, runTitle(c, cfg.Check), serverGroups(cfg, list), func(ctx context.Context, b *tui.Bridge) error {
		run := onScreen(cfg, b)
		var err error
		changed, err = role.Run(ctx, run, list)
		if err == nil { // A plain run prints the summary after the screen, and the log holds it.
			b.Log(role.Summary(changed, cfg.Check))
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return changed, nil
}

// serverGroups names the steps of a run, so that its screen lists each one before the first starts.
func serverGroups(cfg role.Config, list []role.Role) []tui.Group {
	groups := []tui.Group{tui.NewGroup(role.Prepare, cfg.Prepare)}
	for _, r := range list {
		groups = append(groups, tui.NewGroup(r.Name, cfg.Steps(r)))
	}
	return groups
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
