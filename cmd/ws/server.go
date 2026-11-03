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
	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/internal/server/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

const serverUsage = "usage: ws server COMMAND\n\nCommands:\n  fans [CONFIG]  drive the board's fans on the curves of CONFIG, /etc/ws-fans.json, until SIGTERM\n  apply [--check] [--only NAME,...] [--secrets FILE]  set the server up, each role as Go steps\n\n--check plans the steps and applies none. --only runs the named roles. --secrets reads an age-encrypted YAML file with the secrets; the X25519 identity comes on stdin or at the terminal. The last line\nnames the roles that changed.\n"

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
		return fmt.Sprintf("ws server apply is for Ubuntu 26.04, not %s %s", id, version)
	}
	return ""
}

// serverCommand runs the commands of ws server, the server's side of ws. Only its entry in the
// table of groups is built for Linux alone: this function is built everywhere, so that the tests
// run anywhere.
func serverCommand(ctx context.Context, args []string, euid int, m effect.Machine,
	stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		_, _ = io.WriteString(stdout, serverUsage)
		return 0
	}
	if args[0] == "fans" {
		return fansCommand(ctx, args[1:], stderr)
	}
	opts, err := apply.Parse(args[1:])
	var list []roles.Role
	if err == nil && args[0] == "apply" {
		list, err = apply.Select(roles.List(), opts.Only)
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
	// ca-certificates serves the steps' downloads over HTTPS.
	prepare := []step.Step{host.Group{Title: "Packages", Parts: []host.Resource{host.Packages{
		M: m, Out: out, Names: []string{"ca-certificates"},
	}}}}
	v, err := values.Load(m, serverVars)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ws server apply: %v\n", err)
		return 1
	}
	sec, err := loadSecrets(m, opts.Secrets, stdin, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ws server apply: %v\n", err)
		return 1
	}
	changed, err := apply.Run(ctx, apply.Config{Options: opts, Values: v, Secrets: sec, M: m, Out: out, Prepare: prepare}, list)
	term.Flush()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ws server apply: %v\n", err)
		return 1
	}
	out.Say("%s", apply.Summary(changed, opts.Check))
	return 0
}

// loadSecrets reads the identity from stdin and decrypts the file, or returns no secrets where path
// is empty.
func loadSecrets(m effect.Machine, path string, stdin io.Reader, prompt io.Writer) (secrets.Secrets, error) {
	if path == "" {
		return secrets.Secrets{}, nil
	}
	id, err := secrets.ReadIdentity(stdin, prompt)
	if err != nil {
		return secrets.Secrets{}, err
	}
	return secrets.Load(m, path, id)
}
