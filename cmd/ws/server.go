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
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server"
	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/internal/server/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

const serverUsage = "usage: ws server COMMAND\n\nCommands:\n  fans [CONFIG]  drive the board's fans on the curves of CONFIG, /etc/ws-fans.json, until SIGTERM\n  apply [--check] [--only NAME,...] [--secrets FILE] [-- ARG...]  set the server up, each role as Go steps or by the playbook that this ws embeds\n\n--check plans the steps and passes --check --diff to ansible-playbook. --only runs the named roles. --secrets reads an age-encrypted YAML file with the secrets; the X25519 identity comes on stdin or at the terminal. ARG... go to\nansible-playbook; -C, --check, -t and --tags are refused there. The last line names the roles that changed.\n"

const (
	serverLib  = "/var/lib/ws"
	serverVars = "/etc/ws/host_vars.yml"
	serverLog  = "/var/log/ws-ansible.log"
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

// serverCommand runs the commands of ws server, the server's side of ws. Only its entry in the
// table of groups is built for Linux alone: this function is built everywhere, so that the tests
// run anywhere. playbook is the playbook that this ws embeds and version is what ws version prints.
func serverCommand(ctx context.Context, args []string, euid int, playbook fs.FS, version string, m effect.Machine,
	stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		_, _ = io.WriteString(stdout, serverUsage)
		return 0
	}
	if args[0] == "fans" {
		return fansCommand(ctx, args[1:], stderr)
	}
	opts, err := apply.Parse(args[1:])
	var segs []apply.Segment
	if err == nil && args[0] == "apply" {
		segs, err = apply.Segments(roles.List(), opts.Only)
	}
	if args[0] != "apply" || err != nil {
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "ws server apply: %v\n", err)
		}
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
	// What the playbook needs. ansible-core's recommends bring Ubuntu's ansible package, whose
	// collections would stand in for any requirements.yml lacks. python3-apt is named because
	// Ansible's apt module installs it itself only in a real run, not a dry one. ca-certificates
	// serves the playbook's downloads over HTTPS.
	prepare := []step.Step{
		host.Group{Title: "Packages", Parts: []host.Resource{host.Packages{
			M: m, Out: out, Names: []string{"ansible-core", "ca-certificates", "python3-apt"},
		}}},
		server.Playbook{FS: playbook, Version: version, Lib: serverLib, Vars: serverVars, M: m, Out: out},
	}
	v, err := values.Load(m, serverVars, apply.ExtraVars(opts.Args)...)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ws server apply: %v\n", err)
		return 1
	}
	sec, vars, err := loadSecrets(m, opts.Secrets, stdin, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ws server apply: %v\n", err)
		return 1
	}
	changed, err := apply.Run(ctx, apply.Config{Options: opts, Values: v, Secrets: sec, Vars: vars, M: m, Out: out, Prepare: prepare,
		Dir: serverLib + "/" + version, Log: serverLog}, segs)
	term.Flush()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ws server apply: %v\n", err)
		return 1
	}
	out.Say("%s", apply.Summary(changed, opts.Check))
	return 0
}

// loadSecrets reads the identity from stdin and decrypts the file, or returns no secrets where path
// is empty. vars are the secrets as the playbook's extra variables.
func loadSecrets(m effect.Machine, path string, stdin io.Reader, prompt io.Writer) (secrets.Secrets, []byte, error) {
	if path == "" {
		return secrets.Secrets{}, nil, nil
	}
	id, err := secrets.ReadIdentity(stdin, prompt)
	if err != nil {
		return secrets.Secrets{}, nil, err
	}
	s, err := secrets.Load(m, path, id)
	if err != nil {
		return secrets.Secrets{}, nil, err
	}
	vars, err := s.Vars()
	return s, vars, err
}
