package cli

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/internal/keys"
	"github.com/zadykian/ws/roles/client/values"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

const (
	// secretiveApp is where mise's pin puts Secretive (roles/client/macos/ssh/mise.mac.toml).
	secretiveApp = "/Applications/Secretive.app"
	// spctlScript has spctl say its verdict on stdout, as it writes it to stderr.
	spctlScript = `spctl -a -vv "$1" 2>&1`
	// secretiveAgent is Secretive's agent socket as values.yml names it.
	secretiveAgent = "~/" + sshcfg.SecretiveData + "/socket.ssh"
	// pinnedTeam is the team that signs the Secretive.zip of ssh's pin: the OU of the leaf
	// certificate in its code signature. A notarized app of another team is not Secretive.
	pinnedTeam = "Z72PRUAWF6"
)

// secretiveKey is the step that gives the Mac its first key where values.yml names no login key.
// It installs Secretive by its pin, waits for a P-256 key in it, and names that key in values.yml
// for ws's login and git's signing. It plans nothing where values.yml names a login key.
type secretiveKey struct {
	M    effect.Machine
	Out  effect.Out
	Home string
	// Values is values.yml's path, and Conf what it names now.
	Values string
	Conf   values.Conf
	// Pick is --key: the key to take, by its name or fingerprint. Empty takes the one key that
	// comes during the wait.
	Pick string
	// Poll is the wait between two looks at Secretive's agent.
	Poll time.Duration
}

type nameKey struct{ s secretiveKey }

func (c nameKey) String() string {
	return "install Secretive, wait for a key made in it and name the key in " + c.s.Values
}

func (nameKey) Kind() step.ChangeKind { return step.Install }

// Plan finds the work where values.yml names no login key.
func (s secretiveKey) Plan(context.Context) ([]resource.Change, error) {
	if s.Conf.IdentityPubKey != "" {
		return nil, nil
	}
	return []resource.Change{nameKey{s}}, nil
}

// Apply installs Secretive by mise's pin, checks and opens it, waits for the key and names it.
func (c nameKey) Apply(ctx context.Context) error {
	s := c.s
	if err := runMise(ctx, s.M, "bootstrap", "packages", "apply", "--yes", "macos-app:secretive"); err != nil {
		return fmt.Errorf("installing Secretive: %w", err)
	}
	if err := s.notarized(ctx); err != nil {
		return err
	}
	if _, err := s.M.Run(ctx, "open", secretiveApp); err != nil {
		return fmt.Errorf("opening Secretive: %w", err)
	}
	k, err := s.wait(ctx)
	if err != nil {
		return err
	}
	return s.name(k)
}

// notarized refuses an app that macOS does not accept as a notarized Developer ID one of
// Secretive's team.
func (s secretiveKey) notarized(ctx context.Context) error {
	out, err := s.M.Run(ctx, "sh", "-c", spctlScript, "sh", secretiveApp)
	verdict := strings.TrimSpace(string(out))
	lines := strings.Split(verdict, "\n")
	ours := slices.ContainsFunc(lines, func(line string) bool {
		return strings.HasPrefix(line, "origin=Developer ID Application: ") && strings.HasSuffix(line, " ("+pinnedTeam+")")
	})
	if err != nil || !slices.Contains(lines, secretiveApp+": accepted") ||
		!slices.Contains(lines, "source=Notarized Developer ID") || !ours {
		return fmt.Errorf("macOS does not accept %s as a notarized Developer ID app of Secretive's team, %s, so ws "+
			"will not run it (spctl says %q): remove it, and run setup again to install the pin",
			secretiveApp, pinnedTeam, verdict)
	}
	return nil
}

// wait looks at Secretive's agent until it lists the key to take, and returns it. It takes the P-256
// key that --key names, or else the one that came after the agent first answered. ws picks no key
// that was there before, as one made for another use would go to GitHub. A stop of the context ends
// the wait.
func (s secretiveKey) wait(ctx context.Context) (sshcfg.AgentKey, error) {
	if s.Pick != "" {
		s.Out.Say("waiting for the key %s in Secretive, which is open: press Ctrl-C to stop", s.Pick)
	} else {
		s.Out.Say("waiting for a P-256 key in Secretive, which is open: make one at Notify, as it signs git's " +
			"commits too, or press Ctrl-C to stop")
	}
	var before map[string]bool
	for {
		all, socket, err := keys.SecretiveKeys(ctx, s.M, s.Home)
		if socket && err == nil {
			k, found, err := s.take(all, &before)
			if err != nil || found {
				return k, err
			}
		}
		select {
		case <-ctx.Done():
			return sshcfg.AgentKey{}, fmt.Errorf("stopped before Secretive held a key to name: %w", ctx.Err())
		case <-time.After(s.Poll):
		}
	}
}

// take looks for the key to take among those the agent lists. The first look sets before to the
// keys that were there, and says which of them --key could take.
func (s secretiveKey) take(all []sshcfg.AgentKey, before *map[string]bool) (key sshcfg.AgentKey, found bool, err error) {
	p256 := slices.DeleteFunc(slices.Clone(all), func(k sshcfg.AgentKey) bool { return !k.P256() })
	if s.Pick != "" {
		i := slices.IndexFunc(all, func(k sshcfg.AgentKey) bool { return named(k, s.Pick) })
		switch {
		case i < 0:
			if *before == nil {
				*before = map[string]bool{}
				held := "no P-256 key"
				if len(p256) > 0 {
					held = "these P-256 keys: " + describe(p256)
				}
				s.Out.Say("Secretive holds no key that %s names yet, and holds %s", s.Pick, held)
			}
			return key, false, nil
		case !all[i].P256():
			return key, false, fmt.Errorf("the key %s of Secretive's is not a P-256 one, which ws's OpenSSH reads "+
				"alone: make a P-256 key, and run setup again", all[i].Label())
		}
		return all[i], true, nil
	}
	if *before == nil {
		*before = map[string]bool{}
		for _, k := range all {
			(*before)[k.Name] = true
		}
		if len(p256) > 0 {
			s.Out.Say("Secretive holds P-256 keys already, which setup takes only by --key NAME: %s", describe(p256))
		}
		return key, false, nil
	}
	fresh := slices.DeleteFunc(p256, func(k sshcfg.AgentKey) bool { return (*before)[k.Name] })
	switch len(fresh) {
	case 0:
		return key, false, nil
	case 1:
		return fresh[0], true, nil
	}
	return key, false, fmt.Errorf("%d keys came at once, and ws picks none by itself: run setup again with --key NAME "+
		"for one of them: %s", len(fresh), describe(fresh))
}

// keptKey refuses --key where values.yml at file names another login key already. The key it
// names passes, as a run with no --key does, so that the run that named it may run again after a
// failed bootstrap. A pick that is not the key's name is compared with its fingerprint, where
// setup reads the key.
func keptKey(ctx context.Context, m effect.Machine, conf values.Conf, home, file, pick string) error {
	pub := conf.IdentityPubKey
	if strings.TrimSuffix(path.Base(pub), ".pub") == strings.TrimSuffix(pick, ".pub") {
		return nil
	}
	why := "names another key"
	k, err := keys.ReadKey(ctx, m, conf, home, keys.LoginKey)
	switch {
	case err != nil:
		why = "is not its name, and setup cannot read the key to compare fingerprints"
	case sshcfg.AgentKey{Line: strings.TrimSpace(k.Text)}.Fingerprint() == pick:
		return nil
	}
	return fmt.Errorf("%s names WS_IDENTITY_PUB_KEY already, %s, and --key %s %s: run setup with no --key, "+
		"which keeps it", file, pub, pick, why)
}

// named reports whether pick names the key: its name with or without .pub, or its fingerprint.
func named(k sshcfg.AgentKey, pick string) bool {
	return strings.TrimSuffix(k.Name, ".pub") == strings.TrimSuffix(pick, ".pub") || k.Fingerprint() == pick
}

// describe lists the keys by the name that --key takes, and each one's comment and fingerprint.
func describe(list []sshcfg.AgentKey) string {
	var parts []string
	for _, k := range list {
		parts = append(parts, strings.TrimSuffix(k.Name, ".pub")+" for "+k.Label())
	}
	return strings.Join(parts, "; ")
}

// name writes the key into values.yml, through sudo where root owns the file. Setup keeps a value
// that values.yml holds already, and writes the others.
func (s secretiveKey) name(k sshcfg.AgentKey) error {
	pairs := []string{"WS_IDENTITY_PUB_KEY", k.Pub()}
	if s.Conf.IdentityAgent == "" {
		pairs = append(pairs, "WS_IDENTITY_AGENT", secretiveAgent)
	}
	if s.Conf.GitSigningPubKey == "" {
		pairs = append(pairs, "WS_GIT_SIGNING_PUB_KEY", k.Pub())
	}
	text, err := s.M.ReadFile(s.Values)
	if err != nil {
		return err
	}
	info, err := s.M.Stat(s.Values)
	if err != nil {
		return err
	}
	next := values.Set(string(text), pairs...)
	if _, errs := values.MacFile.Parse(next); len(errs) > 0 {
		return errors.New(errs[0].In(s.Values))
	}
	if err := s.M.WriteAtomic(s.Values, []byte(next), effect.Attrs{UID: info.UID, GID: info.GID, Mode: info.Mode.Perm()}); err != nil {
		return err
	}
	for i := 0; i < len(pairs); i += 2 {
		s.Out.Say("set %s in %s to %s", pairs[i], s.Values, pairs[i+1])
	}
	return nil
}
