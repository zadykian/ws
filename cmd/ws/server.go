package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

const (
	serverVars = "/etc/ws/host_vars.yml"
	osRelease  = "/etc/os-release"
)

// serverRefusal returns why ws server apply does not run here, or "" where it does: as root on
// Ubuntu 26.04, the one system that the steps are written for. osRelease is /etc/os-release.
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

// newServerCmd returns ws server, the server's side of ws. It is built everywhere, so that the
// tests run anywhere. Only its entry in the groups is built for Linux alone.
func newServerCmd(p process) *cobra.Command {
	return group("server", "the server's commands", newApplyCmd(p), newFansCmd(p))
}

// newApplyCmd returns ws server apply.
func newApplyCmd(p process) *cobra.Command {
	var opts apply.Options
	c := &cobra.Command{
		Use:   "apply",
		Short: "set the server up, each role as Go steps",
		Long: "--secrets reads an age-encrypted TOML file with the secrets. The X25519 identity comes from the " +
			"unit's credential ws-secrets (systemd's LoadCredential), in $CREDENTIALS_DIRECTORY.\n\n" +
			"The last line names the roles that changed.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error { return runApply(c, p, opts) },
	}
	f := c.Flags()
	f.BoolVar(&opts.Check, "check", false, "plan the steps and apply none")
	f.StringSliceVar(&opts.Only, "only", nil, "run the named roles alone, as a list or by repeating the flag")
	f.StringVar(&opts.Secrets, "secrets", "", "read the secrets from the age-encrypted `FILE`")
	return c
}

// runApply sets the server up as opts say.
func runApply(c *cobra.Command, p process, opts apply.Options) error {
	if c.Flags().Changed("only") && len(opts.Only) == 0 {
		return usageError{errors.New("--only names no role")}
	}
	list, err := apply.Select(roles.List(), opts.Only)
	if err != nil {
		return usageError{err}
	}
	text, err := p.machine.ReadFile(osRelease)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if why := serverRefusal(p.euid, text); why != "" {
		return errors.New(why)
	}
	term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
	out := progOut{term, c.CommandPath(), map[string]bool{}}
	v, err := values.Load(p.machine, serverVars)
	if err != nil {
		return err
	}
	// ws_home is known now: the machine's writes under it stay inside it.
	if m, ok := p.machine.(effect.OSMachine); ok {
		m.Confine = v.Home
		p.machine = m
	}
	sec, err := loadSecrets(p.machine, opts.Secrets, p.getenv)
	if err != nil {
		return err
	}
	changed, err := apply.Run(c.Context(), apply.Config{Options: opts, Values: v, Secrets: sec, M: p.machine, Out: out,
		Observer: out, Prepare: prepareSteps(p.machine, out)}, list)
	term.Flush()
	if err != nil {
		return err
	}
	out.Say("%s", apply.Summary(changed, opts.Check))
	return nil
}

// prepareSteps is what every role needs: ca-certificates serves the steps' downloads over HTTPS.
func prepareSteps(m effect.Machine, out effect.Out) []step.Step {
	return []step.Step{host.Group{Title: "Packages", Parts: []host.Resource{host.Packages{
		M: m, Out: out, Names: []string{"ca-certificates"},
	}}}}
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
	return secrets.Load(m, path, id)
}
