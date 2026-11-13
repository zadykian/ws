// Package cli is the server's side of ws: its Target and its commands, under ws server ubuntu.
package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/role"
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

// Command returns ws server ubuntu, the commands that set up an Ubuntu server.
func Command(p enginecli.Env) *cobra.Command {
	return enginecli.Group("ubuntu", "the commands for an Ubuntu server", newApplyCmd(p), newStateCmd(p), newFansCmd(p),
		newValuesCmd(p), newSecretsCmd(p), newServerWizardCmd(p), newServerConfigureCmd(p))
}

// applyOptions are apply's flags: the runner's, and the --secrets path, which the runner never reads.
type applyOptions struct {
	role.Options
	// Secrets is the age-encrypted file of --secrets, or empty.
	Secrets string
}

// newApplyCmd returns ws server ubuntu apply.
func newApplyCmd(p enginecli.Env) *cobra.Command {
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
		RunE: enginecli.Apply(serverTarget(p, &opts), p),
	}
	f := c.Flags()
	f.Bool("json", false, "print each event of the run as a line of JSON on stdout")
	f.Bool("check", false, "plan the steps and apply none")
	f.StringSliceVar(&opts.Only, "only", nil, "run the named roles and ROLE/STEPs alone, as a list or by repeating the flag")
	f.StringVar(&opts.Secrets, "secrets", "", "read the secrets from the age-encrypted `FILE`")
	return c
}
