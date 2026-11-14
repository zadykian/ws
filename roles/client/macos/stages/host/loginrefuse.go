package host

import (
	"context"
	"fmt"
	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"strings"
)

// refuse says why WS_USER refuses the Mac's key, and what to do, or under --check that a run would.
func (r *Login) refuse(ctx context.Context, w remote.Way, sshErr string) error {
	if strings.Contains(sshErr, "Host key verification failed") {
		return fmt.Errorf("%s: %s at %s:%s offers another host key than known_hosts holds under %[2]s: where %[2]s was "+
			"installed again, run ssh-keygen -R %[2]s, then the wizard again", stage.Login, r.SSH.Host, w.Host, w.Port)
	}
	if sshErr != "" {
		sshErr = ". ssh's error: " + sshErr
	}
	if r.Check {
		r.Out.Say("%s: %s on %s doesn't log in by the Mac's key; a run would stop, as the wizard adds no key "+
			"to %s%s", stage.Login, w.User, r.SSH.Host, r.SSH.Host, sshErr)
		return nil
	}
	pub, _, err := r.Gh.Key(ctx)
	if err != nil {
		return fmt.Errorf("%s: the Mac's key for %s has no .pub", stage.Login, r.SSH.Host)
	}
	return fmt.Errorf("%s: %s on %s doesn't log in by the Mac's key, %s, and the wizard adds no key to %[3]s. %[3]s takes "+
		"its keys from the GitHub account at its install: Ubuntu's installer imports them for its user, and on a "+
		"cloud VM, cloud-init gives its default user the provider's keys, which must be the account's. Set WS_USER "+
		"in server.conf to that user, then run the wizard again%[5]s", stage.Login, w.User, r.SSH.Host, pub, sshErr)
}
