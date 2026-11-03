package network

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/effect"
)

// netplanDirs are the directories whose files netplan merges.
var netplanDirs = []string{"lib/netplan", "etc/netplan", "run/netplan"}

// checkNetplan has netplan merge its files with the direct link's in a scratch root, and returns
// what keeps it from setting up the port alone. netplan generate, run in a snap's shell, ignores
// --root-dir and generates the machine's files over D-Bus, so the step runs configure, which
// writes under the root only. The settings may hold secrets, such as Wi-Fi passwords, so no
// message quotes what netplan prints, and the root is gone as the check ends.
func checkNetplan(ctx context.Context, m effect.Machine, file []byte, port Port) ([]string, error) {
	made, err := m.RunRoot(ctx, "mktemp", "-d", "--suffix=.netplan")
	if err != nil {
		return nil, fmt.Errorf("making a scratch root for netplan: %w", err)
	}
	root := strings.TrimSpace(string(made))
	defer func() { _, _ = m.RunRoot(context.WithoutCancel(ctx), "rm", "-rf", "--", root) }()
	if _, err := m.RunRoot(ctx, "mkdir", "-p", "--", root+"/etc/netplan"); err != nil {
		return nil, err
	}
	for _, dir := range netplanDirs {
		if info, err := m.Stat("/" + dir); err != nil || !info.Mode.IsDir() {
			continue
		}
		if _, err := m.RunRoot(ctx, "mkdir", "-p", "--", root+"/"+dir); err != nil {
			return nil, err
		}
		if _, err := m.RunRoot(ctx, "cp", "-a", "/"+dir+"/.", root+"/"+dir+"/"); err != nil {
			return nil, fmt.Errorf("copying netplan's files into the scratch root: %w", err)
		}
	}
	if err := m.WriteRootFile(root+netplanPath, file, 0o600); err != nil {
		return nil, err
	}
	generated := runRoot(ctx, m, "/usr/libexec/netplan/configure", "--root-dir", root)
	got := runRoot(ctx, m, "netplan", "get", "--root-dir", root)
	return NetplanProblems(generated, got, root, port), nil
}
