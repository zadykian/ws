package wizard

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zadykian/ws/mac/internal/setup"
	"github.com/zadykian/ws/step"
)

// Apply makes what the plan lists, in order.
func (k *MacKey) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		var err error
		switch c := c.(type) {
		case makeKey:
			err = k.make(ctx, c.path)
		case setup.Write:
			err = k.pin(ctx, c)
		case chooseKey:
			err = k.choose(ctx, c)
		case gitKey:
			err = k.git(ctx, false)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// make makes the ed25519 key, whose passphrase ssh-keygen asks at the terminal. It gives the key to
// ssh's agent, and the passphrase to the Keychain, so that ssh asks no more. Root's key logs in with
// BatchMode, where ssh asks for no passphrase, so a passphrase that neither holds fails it. Where
// ssh-add fails, the stage stops. The next run finds the key there and makes no more, so the stop
// names the command to run by hand.
func (k *MacKey) make(ctx context.Context, key string) error {
	k.Out.Say("%s: making %s: ssh-keygen asks its passphrase twice, then ssh-add once", macKey, key)
	if err := k.M.Pass(ctx, false, "ssh-keygen", "-q", "-t", "ed25519", "-a", "100", "-f", key); err != nil {
		return fmt.Errorf("%s: ssh-keygen cannot make %s", macKey, key)
	}
	if err := k.M.Pass(ctx, false, "ssh-add", "--apple-use-keychain", key); err != nil {
		return fmt.Errorf("%s: %s is made, but ssh-add cannot give it to the agent: run ssh-add "+
			"--apple-use-keychain %s, which asks its passphrase, then run the wizard again", macKey, key, key)
	}
	return nil
}

// pin adds the line that the write holds to mac.conf, so that a Secretive key made later does not
// take the place of a file key.
func (k *MacKey) pin(ctx context.Context, w setup.Write) error {
	after := strings.TrimSuffix(string(w.After), "\n")
	return refused(k.file().add(ctx, after[strings.LastIndex(after, "\n")+1:]+"\n"))
}

// refused words the stop where mac.conf with a new line would be refused.
func refused(err error) error {
	if errors.Is(err, errRefused) {
		return fmt.Errorf("%s: ssh has no usable key for ws, as above; mend that, then run the wizard again", macKey)
	}
	return err
}
