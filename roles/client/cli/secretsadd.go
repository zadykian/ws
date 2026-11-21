package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os/signal"
	"path"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/secrets"
	"github.com/zadykian/ws/roles/client/values"
)

// serverSecretsRun runs the server's task secrets, which fetches the secrets repository and runs
// the bootstrap with the secrets, as root, by the server's own mise.
var serverSecretsRun = []string{"sudo", "env", "MISE_LOCKED=1", "mise", "-C", "/opt/ws/roles/server", "run",
	"secrets"}

// newSecretsAddCmd returns ws client secrets add.
func newSecretsAddCmd(p enginecli.Env, server *string) *cobra.Command {
	var address string
	var port int
	c := &cobra.Command{
		Use:   "add NAME",
		Short: "add a server's host key to the secrets, and run the bootstrap with them there",
		Long: "It takes the ed25519 host key of the server NAME from known_hosts. Where known_hosts holds none, it " +
			"scans the server, prints the key's fingerprint, and records the key once you say that the server's " +
			"console prints the same. It reads the server's hostname by ssh. In a temporary clone of the secrets " +
			"repository, it makes the key the hostname's recipient in fnox.toml and in fnox.HOSTNAME.toml, and " +
			"encrypts their values again, which asks for the Nano's PIN and a touch. It keeps the key as the " +
			"repository's read-only deploy key, titled with the hostname, and lands the change by a pull request " +
			"once its checks pass. It then runs the server's task secrets over ssh -t. Last, it prints each host key " +
			"that the secrets hold and known_hosts gives no host, to drop by hand where no server holds it now. " +
			"Each step changes nothing where it is done, so run it again where one fails.",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			// A Ctrl-C or a SIGTERM ends the context, not ws, so that add still removes its clone.
			ctx, stop := signal.NotifyContext(c.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			c.SetContext(ctx)
			return runSecretsAdd(c, p, *server, args[0], address, port)
		},
	}
	c.Flags().StringVar(&address, "address", "", "the server's address, where ssh reaches it before the tunnel is up")
	c.Flags().IntVar(&port, "port", 0, "the server's ssh port at --address, where it is not 22")
	return c
}

// runSecretsAdd adds the host key of the server called name to the secrets, and runs them there.
func runSecretsAdd(c *cobra.Command, p enginecli.Env, server, name, address string, port int) error {
	if p.Euid == 0 {
		return errors.New("run it as yourself, not root: the Nano, fnox's key file and gh's login are yours")
	}
	if server != "" {
		return errors.New("add takes the server's name alone")
	}
	if port < 0 || port > 65535 {
		return fmt.Errorf("--port %d is not a port", port)
	}
	if err := knownServer(p, name); err != nil {
		return err
	}
	keyFile, err := secrets.KeyFile(p.Getenv)
	if err != nil {
		return err
	}
	if _, err := p.Machine.Stat(keyFile); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("no fnox key file %s: run ws client secrets init first, which keeps the Nano's "+
			"stub there", keyFile)
	} else if err != nil {
		return fmt.Errorf("cannot stat fnox's key file %s: %w", keyFile, err)
	}
	target := sshTarget(name, address, port)
	key, err := hostKey(c, p, name, target)
	if err != nil {
		return err
	}
	hostname, err := hostnameOf(c.Context(), p.Machine, name, target)
	if err != nil {
		return err
	}
	byHand, err := addHostKey(c, p, name, hostname, key, keyFile)
	if err != nil {
		return err
	}
	again := strings.Join(append([]string{"ws client secrets add"}, addFlags(name, address, port)...), " ")
	_, _ = fmt.Fprintf(c.OutOrStdout(), "running the bootstrap with the secrets on %s, which takes 20 to 40 "+
		"minutes on a new server\n", name)
	run := append(append([]string{"-t"}, target...), serverSecretsRun...)
	if err = p.Machine.Pass(c.Context(), false, "ssh", run...); err != nil {
		err = fmt.Errorf("the run of the secrets on %s: %w: run %s again", name, err, again)
	}
	// The lines come after the run's long output, so that they stay in sight.
	for _, line := range byHand {
		_, _ = fmt.Fprintln(c.OutOrStdout(), line)
	}
	return err
}

// knownServer refuses a name that is no server of the Mac's.
func knownServer(p enginecli.Env, name string) error {
	if !values.IsName(name) {
		return fmt.Errorf("%s is not a server's name: %s", name, values.NameForm)
	}
	mac := values.Path(p.Getenv, p.Pwd)
	names, err := values.Servers(p.Machine, mac)
	if err != nil {
		return err
	}
	if !slices.Contains(names, name) {
		return fmt.Errorf("no server named %s in %s, which holds %s", name,
			path.Dir(path.Dir(values.ServerPath(mac, name))), values.List(names))
	}
	return nil
}

// sshTarget is ssh's arguments that reach the server name: at address and port, where given,
// before the tunnel is up, else as the Mac's ssh config says.
func sshTarget(name, address string, port int) []string {
	var target []string
	if address != "" {
		target = append(target, "-o", "HostName="+address)
	}
	if port != 0 {
		target = append(target, "-o", "Port="+strconv.Itoa(port))
	}
	return append(target, name)
}

// addFlags are add's arguments for the server name at address and port, as a rerun takes them.
func addFlags(name, address string, port int) []string {
	args := []string{name}
	if address != "" {
		args = append(args, "--address", address)
	}
	if port != 0 {
		args = append(args, "--port", strconv.Itoa(port))
	}
	return args
}

// hostKey is the server's ed25519 host key that known_hosts holds under name. Where it holds none,
// it scans the server at ssh's address. It records the scanned key once the user says that the
// server's console prints the same fingerprint, as nothing else proves the key the server's.
func hostKey(c *cobra.Command, p enginecli.Env, name string, target []string) (string, error) {
	ctx, m, home := c.Context(), p.Machine, p.Getenv("HOME")
	if home == "" {
		return "", errors.New("HOME is not set: known_hosts has no place")
	}
	if key, err := secrets.HostKey(ctx, m, home, name); err != nil || key != "" {
		return key, err
	}
	host, port, err := sshAddress(ctx, m, target)
	if err != nil {
		return "", err
	}
	key, err := secrets.ScanHostKey(ctx, m, host, port)
	if err != nil {
		return "", err
	}
	fingerprint, err := secrets.Fingerprint(ctx, m, key)
	if err != nil {
		return "", err
	}
	_, _ = fmt.Fprintf(c.ErrOrStderr(), "%s holds no host key of %s. The server at %s:%d offers the key\n\n    %s\n\n"+
		"At the server's console, ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub prints its own.\n"+
		"Does the console print the same fingerprint? (yes/no) [no]: ", secrets.KnownHosts(home), name, host, port,
		fingerprint)
	answer, err := readLine(ctx, c.InOrStdin())
	if err != nil {
		return "", fmt.Errorf("no answer on the host key of %s, so nothing changed: %w", name, err)
	}
	if answer != "yes" {
		return "", fmt.Errorf("the host key of %s is not the console's, so nothing changed", name)
	}
	if err := secrets.RecordHostKey(m, home, name, key); err != nil {
		return "", err
	}
	_, _ = fmt.Fprintf(c.ErrOrStderr(), "recorded the host key of %s in %s\n", name, secrets.KnownHosts(home))
	return key, nil
}

// readLine reads a line of in, trimmed. ctx's end stops the wait, as a Ctrl-C reaches ws as ctx's
// end alone, and the read stays blocked past it until the process ends.
func readLine(ctx context.Context, in io.Reader) (string, error) {
	type reply struct {
		line string
		err  error
	}
	read := make(chan reply, 1)
	go func() {
		line, err := bufio.NewReader(in).ReadString('\n')
		if errors.Is(err, io.EOF) && line != "" {
			err = nil
		}
		read <- reply{strings.TrimSpace(line), err}
	}()
	select {
	case r := <-read:
		return r.line, r.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// sshAddress is the host and port that ssh reaches target at, from ssh -G's lines, which hold the
// config's values under the options' own.
func sshAddress(ctx context.Context, m effect.Machine, target []string) (string, int, error) {
	out, err := m.Run(ctx, "ssh", append([]string{"-G"}, target...)...)
	if err != nil {
		return "", 0, fmt.Errorf("ssh -G cannot read the address of %s: %w", target[len(target)-1], err)
	}
	var host string
	port := 0
	for line := range strings.Lines(string(out)) {
		key, value, _ := strings.Cut(strings.TrimSpace(line), " ")
		switch key {
		case "hostname":
			host = value
		case "port":
			if port, err = strconv.Atoi(value); err != nil {
				return "", 0, fmt.Errorf("ssh -G gives the port %q", value)
			}
		}
	}
	if host == "" || port == 0 {
		return "", 0, fmt.Errorf("ssh -G gives no hostname and port of %s", target[len(target)-1])
	}
	return host, port, nil
}

// hostnameOf is the server's hostname, which names its profile, label and deploy key.
func hostnameOf(ctx context.Context, m effect.Machine, name string, target []string) (string, error) {
	out, err := m.Run(ctx, "ssh", append(slices.Clone(target), "hostname")...)
	if err != nil {
		return "", fmt.Errorf("cannot read the hostname of %s by ssh: %w: where ssh names another host key, as "+
			"after a rebuild, drop the old one by ssh-keygen -R %s, then run add again", name, err, name)
	}
	hostname := strings.TrimSpace(string(out))
	if err := secrets.CheckHostname(hostname); err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return hostname, nil
}
