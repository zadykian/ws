// Package cli is the server's side of ws: its Target and its commands, under ws server ubuntu.
package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/units"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/roles/server/ubuntu/ikeward"
)

const (
	serverVars = "/etc/ws/values.yml"
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
		newCableCmd(p, units.OSUnits{}), newTunnelCmd(p, units.OSUnits{}), newIkewardCmd(p, ikeward.StatusTries, ikeward.StatusWait),
		newGitCmd(p), newClaudeCmd(p), newValuesCmd(p), newSecretsCmd(p), newSignozCmd(p))
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
		Short: "install ca-certificates alone, as mise's bootstrap sets each role up",
		Long: "Each role is mise's now, so apply holds no role in Go: it installs ca-certificates alone, and " +
			"--only refuses every name. mise's bootstrap sets the server up, as the server's runbook says.\n\n" +
			"--secrets reads an age-encrypted TOML file with the secrets. The X25519 identity comes from the " +
			"unit's credential ws-secrets (systemd's LoadCredential), in $CREDENTIALS_DIRECTORY.\n\n" +
			"The last line says whether anything changed.\n\n" +
			"--json prints each event of the run as a line of JSON on stdout: planned, started, done, failed " +
			"or skipped, with the step, the first line of each change and the error. ws's own lines stay on " +
			"stderr.",
		Args: cobra.NoArgs,
		RunE: enginecli.Apply(serverTarget(p, &opts), p),
	}
	f := c.Flags()
	f.Bool("json", false, "print each event of the run as a line of JSON on stdout")
	f.Bool("check", false, "plan the steps and apply none")
	f.StringSliceVar(&opts.Only, "only", nil, "a role or ROLE/STEP to run alone, which ws holds none of in Go")
	f.StringVar(&opts.Secrets, "secrets", "", "read the secrets from the age-encrypted `FILE`")
	return c
}
