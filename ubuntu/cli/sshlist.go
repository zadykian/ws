package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/sshkey"
)

// sshStore is the STORE of each key that list shows. A key here is a file of root's. The Mac's
// seward fills the same column.
const sshStore = "file"

// newSSHListCmd returns ws ubuntu ssh list.
func newSSHListCmd(p enginecli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list the keys of root's ~/.ssh, a line each of NAME, TYPE, FINGERPRINT and STORE",
		Long: "It lists each NAME.pub of /root/.ssh whose name passes the rule of ssh keygen, in order, a line " +
			"each with the fields NAME, TYPE, FINGERPRINT as ssh-keygen -l prints it, and STORE, which is file, " +
			"tab-separated, with no header. A .pub with no OpenSSH key beside it, or that cannot be read, gets " +
			"-, - and error, and its reason goes to standard error. list then exits 1, once it has listed " +
			"every key. It refuses a link at /root/.ssh.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			names, err := sshKeyNames(p.Machine)
			if err != nil {
				return err
			}
			failed := false
			for _, name := range names {
				line, err := sshLine(p.Machine, name)
				if err != nil {
					failed = true
					_, _ = fmt.Fprintf(c.OutOrStdout(), "%s\t-\t-\terror\n", name)
					_, _ = fmt.Fprintf(c.ErrOrStderr(), "%s: %s: %v\n", c.CommandPath(), name, err)
					continue
				}
				_, _ = fmt.Fprintln(c.OutOrStdout(), line)
			}
			if failed {
				return engine.ExitStatus{Code: 1}
			}
			return nil
		},
	}
}

// sshKeyNames returns the name of each .pub in ~/.ssh that the rule allows. A directory that is
// missing holds none.
func sshKeyNames(m effect.Machine) ([]string, error) {
	dir := rootUser.Home + "/" + sshDir
	info, err := m.Stat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if info.Link != "" {
		return nil, fmt.Errorf("%s is a link to %s: replace it with the directory itself", dir, info.Link)
	}
	files, err := m.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, f := range files {
		if name, ok := strings.CutSuffix(f, ".pub"); ok && sshkey.Check(name) == nil {
			names = append(names, name)
		}
	}
	return names, nil
}

// sshLine returns the line of the key NAME, or the reason why its files are no key.
func sshLine(m effect.Machine, name string) (string, error) {
	pub, found, err := sshFile(m, name+".pub")
	if err != nil {
		return "", err
	}
	if !found {
		return "", errors.New("the .pub is gone")
	}
	key, found, err := sshFile(m, name)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("no key %s beside the .pub", name)
	}
	if !isPrivateKey(key) {
		return "", fmt.Errorf("%s is no OpenSSH private key", name)
	}
	public, _, _, _, err := ssh.ParseAuthorizedKey(pub)
	if err != nil {
		return "", fmt.Errorf("the .pub holds no public key: %w", err)
	}
	return strings.Join([]string{name, public.Type(), ssh.FingerprintSHA256(public), sshStore}, "\t"), nil
}
