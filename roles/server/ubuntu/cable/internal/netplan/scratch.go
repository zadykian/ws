// Package netplan checks, in a scratch root, that netplan sets up the cable's port alone with its
// file.
package netplan

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

const (
	// Path is the cable's netplan file. Its name sorts after the installer's.
	Path = "/etc/netplan/60-cable.yaml"
	// configure writes networkd's files, and the rest, from netplan's, under a root of its own.
	configure = "/usr/libexec/netplan/configure"
)

// netplanDirs are the directories whose files netplan merges.
var netplanDirs = []string{"lib/netplan", "etc/netplan", "run/netplan"}

// Check has netplan merge its files with the cable's in a scratch root, and returns
// what keeps it from setting up the port alone. netplan generate, run in a snap's shell, ignores
// --root-dir and generates the machine's files over D-Bus. So the step runs configure, which
// writes under the root only. The settings may hold secrets, such as Wi-Fi passwords, so no
// message quotes what netplan prints, and the root is gone as the check ends.
func Check(ctx context.Context, m effect.Machine, file []byte, port Port) ([]string, error) {
	root, done, err := scratchCopy(ctx, m)
	if err != nil {
		return nil, err
	}
	defer done()
	if err := m.WriteRootFile(root+Path, file, 0o600); err != nil {
		return nil, err
	}
	generated := runRoot(ctx, m, configure, "--root-dir", root)
	got := runRoot(ctx, m, "netplan", "get", "--root-dir", root)
	return Problems(generated, got, root, port), nil
}

// scratchCopy makes a scratch root that holds a copy of netplan's files, and returns it with what
// removes it.
func scratchCopy(ctx context.Context, m effect.Machine) (string, func(), error) {
	made, err := m.RunRoot(ctx, "mktemp", "-d", "--suffix=.netplan")
	if err != nil {
		return "", nil, fmt.Errorf("making a scratch root for netplan: %w", err)
	}
	root := strings.TrimSpace(string(made))
	done := func() { _, _ = m.RunRoot(context.WithoutCancel(ctx), "rm", "-rf", "--", root) }
	if err := copyNetplan(ctx, m, root); err != nil {
		done()
		return "", nil, err
	}
	return root, done, nil
}

// copyNetplan copies each of netplanDirs that the machine has into root. It makes
// root+"/etc/netplan" where the machine lacks it.
func copyNetplan(ctx context.Context, m effect.Machine, root string) error {
	if _, err := m.RunRoot(ctx, "mkdir", "-p", "--", root+"/etc/netplan"); err != nil {
		return err
	}
	for _, dir := range netplanDirs {
		if info, err := m.Stat("/" + dir); err != nil || !info.Mode.IsDir() {
			continue
		}
		if _, err := m.RunRoot(ctx, "mkdir", "-p", "--", root+"/"+dir); err != nil {
			return err
		}
		if _, err := m.RunRoot(ctx, "cp", "-a", "/"+dir+"/.", root+"/"+dir+"/"); err != nil {
			return fmt.Errorf("copying netplan's files into the scratch root: %w", err)
		}
	}
	return nil
}
