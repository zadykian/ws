package wizard

import (
	"context"
	"fmt"
	"strings"
)

// refuse says why root refuses the Mac's key, and what to do, or under --check that a run would.
func (r *Login) refuse(ctx context.Context, w Way, sshErr string) error {
	if strings.Contains(sshErr, "Host key verification failed") {
		return fmt.Errorf("%s: ws at %s:%s offers another host key than known_hosts holds under ws: where ws was "+
			"installed again, run ssh-keygen -R ws, then the wizard again", loginStage, w.Host, w.Port)
	}
	if sshErr != "" {
		sshErr = ". ssh's error: " + sshErr
	}
	if r.Host.Check {
		r.Host.Out.Say("%s: root on ws doesn't log in by the Mac's key; a run would stop, as the wizard adds no key "+
			"to ws%s", loginStage, sshErr)
		return nil
	}
	pub, _, err := r.Gh.key(ctx)
	if err != nil {
		return fmt.Errorf("%s: the Mac's key for ws has no .pub", loginStage)
	}
	return fmt.Errorf("%s: root on ws doesn't log in by the Mac's key, %s, and the wizard adds no key to ws. ws takes "+
		"its keys from the GitHub account at its install: Ubuntu's installer imports them for its user, and on a "+
		"cloud VM, cloud-init gives its default user the provider's keys, which must be the account's. Copy that "+
		"user's authorized_keys to root, as the README's \"A new machine\" does: root's file is replaced, so "+
		"cloud-init's forced command goes. Then run the wizard again%s", loginStage, pub, sshErr)
}
