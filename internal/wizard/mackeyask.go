package wizard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/internal/sshcfg"
)

// newKey, the default option, has the user make a key in Secretive's app, so that a keypress never
// takes a key made for another use.
const newKey = "make a new one"

// keyUse is what a key serves: its name in the question, its key in mac.conf, and what to make in
// Secretive's app.
type keyUse struct{ name, key, make string }

var (
	// loginUse is ws's login key, at Secretive's default protection, which asks for Touch ID at
	// each login.
	loginUse = keyUse{"ws's login key", "WS_IDENTITY_PUB_KEY", "a P-256 key for ws's login, at its default protection"}
	// gitUse is git's key, at Notify, which signs with no prompt while the Mac is unlocked (#109).
	gitUse = keyUse{"git's signing key", "WS_GIT_SIGNING_PUB_KEY", "a P-256 key for git's signing, at Notify"}
)

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

// ask lists the agent's keys again, with the login key that an earlier answer may have named. It
// names the key answered in mac.conf, or with no terminal, it stops and names each key's line.
func (k *MacKey) ask(ctx context.Context, use keyUse) error {
	keys, _, err := setup.SecretiveKeys(ctx, k.M, k.Home)
	if err != nil {
		return fmt.Errorf("%s: %w", macKey, err)
	}
	if !k.Tty {
		return noTerminal(use, keys)
	}
	_, c, err := loadConf(k.file())
	if err != nil {
		return err
	}
	login := k.login(c)
	answer, err := effect.Choose(ctx, k.P, keyQuestion(use, keys, login))
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("%s: the terminal ended before the answer", macKey)
	}
	if err != nil {
		return err
	}
	for _, key := range keys {
		if key.P256() && option(key, login) == answer {
			return k.name(ctx, use, key)
		}
	}
	key, err := k.made(ctx, use.make, keys)
	if err != nil {
		return err
	}
	return k.name(ctx, use, key)
}

// noTerminal is the stop where no terminal is there to ask.
func noTerminal(use keyUse, keys []sshcfg.AgentKey) error {
	var lines []string
	for _, key := range keys {
		if key.P256() {
			lines = append(lines, line(use.key, key.Pub())+" for "+key.Label())
		}
	}
	if len(lines) == 0 {
		return fmt.Errorf("%s: no terminal to ask for %s, and Secretive's agent lists no P-256 key: run the wizard "+
			"at a terminal to make one, or name a key's .pub in mac.conf's %s", macKey, use.name, use.key)
	}
	return fmt.Errorf("%s: no terminal to ask for %s: add one of these lines to mac.conf, %s; or run the wizard "+
		"at a terminal", macKey, use.name, strings.Join(lines, "; "))
}

// name names the key in mac.conf, once it plans and writes ws's copy of its .pub, with the
// directories of the copies, which ssh and git read.
// For ws's login key, it names Secretive's agent too, which holds the private key.
func (k *MacKey) name(ctx context.Context, use keyUse, key sshcfg.AgentKey) error {
	copied, err := setup.PlanCopy(k.M, sshcfg.Expand(key.Pub(), k.Home), key.Line+"\n", k.Home)
	if err != nil {
		return err
	}
	for _, c := range copied {
		if err := setup.Apply(ctx, k.M, k.Out, c); err != nil {
			return err
		}
	}
	settings := []setting{{use.key, key.Pub()}}
	if use == loginUse {
		settings = append(settings, setting{"WS_IDENTITY_AGENT", "~/" + sshcfg.SecretiveData + "/socket.ssh"})
	}
	return refused(k.file().setAll(ctx, settings...))
}
