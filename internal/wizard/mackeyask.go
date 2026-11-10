package wizard

import (
	"fmt"
	"path"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/sshcfg"
)

// newKey, the default option, has the user make a key in Secretive's app, so that a keypress never
// takes a key made for another use.
const newKey = "make a new one"

// keyUse is what a key serves: its name in the question, its key in mac.conf, and what to make in
// Secretive's app.
type keyUse struct{ name, key, make string }

// gitUse is git's key, at Notify, which signs with no prompt while the Mac is unlocked (#109).
var gitUse = keyUse{"git's signing key", "WS_GIT_SIGNING_PUB_KEY", "a P-256 key for git's signing, at Notify"}

// askKey is the change that asks which of Secretive's keys serves the use, and names it in mac.conf.
type askKey struct {
	use keyUse
	q   effect.ChoiceQuestion
}

func (a askKey) String() string {
	options := make([]string, len(a.q.Options))
	for i, o := range a.q.Options {
		if o == a.q.Default {
			o += " (the default)"
		}
		options[i] = o
	}
	return fmt.Sprintf("%s: would ask for %s: %s", macKey, a.q.Text, strings.Join(options, "; "))
}

// keyQuestion offers each P-256 key of keys by its label, and a new one, the default. login names
// the .pub of ws's login key, which git's question marks, as one key may serve both.
func keyQuestion(use keyUse, keys []sshcfg.AgentKey, login string) effect.ChoiceQuestion {
	q := effect.ChoiceQuestion{Text: use.name + ", in Secretive", Default: newKey}
	for _, key := range keys {
		if key.P256() {
			q.Options = append(q.Options, option(key, login))
		}
	}
	q.Options = append(q.Options, newKey)
	return q
}

// option is the key's answer, with ws's login key marked.
func option(key sshcfg.AgentKey, login string) string {
	if key.Name == login {
		return key.Label() + ", ws's login key"
	}
	return key.Label()
}

// login names the .pub of mac.conf's key for ws's login, where Secretive holds it, for git's
// question to mark.
func (k *MacKey) login(c conf.Conf) string {
	if pub := sshcfg.Expand(c.IdentityPubKey, k.Home); c.IdentityPubKey != "" && sshcfg.SecretiveKey(pub, k.Home) {
		return path.Base(pub)
	}
	return ""
}
