package cli

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/resource"
)

// newSSHRemoveCmd returns ws ubuntu ssh remove.
func newSSHRemoveCmd(p enginecli.Env) *cobra.Command {
	c := &cobra.Command{
		Use:   "remove NAME",
		Short: "delete root's key NAME and its .pub, after the name is typed again",
		Long: "It deletes whichever of /root/.ssh/NAME and NAME.pub exist, once the name is typed again. --yes " +
			"skips the question. With no terminal and no --yes it refuses as a wrong call, and exits 2. It exits " +
			"1 where neither file exists, where NAME exists and is no OpenSSH private key, or where a link " +
			"stands at a file or at /root/.ssh. Then it deletes nothing.",
		Args: sshNameArg,
	}
	yes := c.Flags().Bool("yes", false, "delete without asking")
	c.RunE = func(c *cobra.Command, args []string) error {
		name := args[0]
		if !*yes && !p.Terminal {
			return engine.UsageError{Err: fmt.Errorf("no terminal to ask on: pass --yes to delete %s", name)}
		}
		key, hasKey, err := sshFile(p.Machine, name)
		if err != nil {
			return err
		}
		_, hasPub, err := sshFile(p.Machine, name+".pub")
		if err != nil {
			return err
		}
		switch {
		case !hasKey && !hasPub:
			return fmt.Errorf("no key %s in %s/%s", name, rootUser.Home, sshDir)
		case hasKey && !isPrivateKey(key):
			return fmt.Errorf("%s/%s/%s is no OpenSSH private key, so nothing is deleted", rootUser.Home, sshDir, name)
		}
		if !*yes {
			if err := confirm(c, name); err != nil {
				return err
			}
		}
		a, err := rootUser.Lookup(c.Context(), p.Machine)
		if err != nil {
			return err
		}
		var gone []resource.Resource
		for _, f := range []struct {
			file string
			held bool
		}{{name, hasKey}, {name + ".pub", hasPub}} {
			if f.held {
				gone = append(gone, a.Absent(p.Machine, sshDir+"/"+f.file))
			}
		}
		out, flush := sshOut(c)
		defer flush()
		return settle(c.Context(), out, gone, false)
	}
	return c
}

// confirm asks for the name again on standard error, and fails unless the answer is the name.
func confirm(c *cobra.Command, name string) error {
	_, _ = fmt.Fprintf(c.ErrOrStderr(), "Delete the key %s and its .pub? Type %s to confirm: ", name, name)
	answer, err := bufio.NewReader(c.InOrStdin()).ReadString('\n')
	if err != nil && answer == "" {
		return errors.New("no answer, so nothing changed")
	}
	if strings.TrimSpace(answer) != name {
		return fmt.Errorf("not confirmed, so %s stays", name)
	}
	return nil
}
