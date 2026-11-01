package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/mac/internal/server"
	"github.com/zadykian/ws/step"
)

const serverUsage = "usage: ws server COMMAND\n\nCommands:\n  apply [-- ARG...]  set the server up from the playbook that this ws embeds; ARG... go to ansible-playbook\n"

const (
	serverLib  = "/var/lib/ws"
	serverVars = "/etc/ws/host_vars.yml"
	osRelease  = "/etc/os-release"
)

// serverRefusal returns why ws server apply does not run here, or "" where it does: as root on
// Ubuntu 26.04, the one system that the playbook is written for. osRelease is /etc/os-release.
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
		return fmt.Sprintf("the playbook is for Ubuntu 26.04, not %s %s", id, version)
	}
	return ""
}

// serverApplyArgs returns the arguments for ansible-playbook: those after "--". Anything else
// before it is not accepted.
func serverApplyArgs(args []string) ([]string, bool) {
	if len(args) == 0 {
		return nil, true
	}
	if args[0] != "--" {
		return nil, false
	}
	return args[1:], true
}

// serverCommand runs the commands of ws server, the server's side of ws. Only its entry in the
// table of groups is built for Linux alone: this function is built everywhere, so that the tests
// run anywhere. playbook is the playbook that this ws embeds and version is what ws version prints.
func serverCommand(ctx context.Context, args []string, euid int, playbook fs.FS, version string, m effect.Machine,
	stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		_, _ = io.WriteString(stdout, serverUsage)
		return 0
	}
	rest, ok := serverApplyArgs(args[1:])
	if args[0] != "apply" || !ok {
		_, _ = io.WriteString(stderr, serverUsage)
		return 2
	}
	text, err := m.ReadFile(osRelease)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		_, _ = fmt.Fprintf(stderr, "ws server apply: %v\n", err)
		return 1
	}
	if why := serverRefusal(euid, text); why != "" {
		_, _ = fmt.Fprintf(stderr, "ws server apply: %s\n", why)
		return 1
	}
	term := effect.NewTerminal(strings.NewReader(""), stderr)
	out := progOut{term, "ws server apply", map[string]bool{}}
	steps := []step.Step{
		server.Packages{M: m, Out: out},
		server.Playbook{FS: playbook, Version: version, Lib: serverLib, Vars: serverVars, M: m, Out: out},
	}
	// Not a check, even under the playbook's --check: ansible-playbook cannot run without them.
	_, err = step.Run(ctx, steps, out, false)
	term.Flush()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ws server apply: %v\n", err)
		return 1
	}
	if err := execPlaybook(m, version, rest); err != nil {
		_, _ = fmt.Fprintf(stderr, "ws server apply: %v\n", err)
		return 1
	}
	// Exec returns only where it failed, or in a fake.
	return 0
}

// execPlaybook replaces ws with ansible-playbook, which runs the version's unpacked site.yml.
// There is no chdir and stdin is untouched: -e @FILE and --vault-password-file /dev/fd/3 keep the
// caller's directory and descriptors.
func execPlaybook(m effect.Machine, version string, rest []string) error {
	dir := serverLib + "/" + version
	argv := append([]string{"-i", dir + "/inventory.ini", dir + "/site.yml"}, rest...)
	env := []string{"LC_ALL=C.UTF-8", "ANSIBLE_CONFIG=" + dir + "/ansible.cfg", "ANSIBLE_COLLECTIONS_PATH=" + dir + "/collections"}
	return m.Exec("ansible-playbook", argv, env)
}
