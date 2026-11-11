package wizard

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/internal/sshcfg"
)

// fresh returns the first key that Secretive's agent lists and before lacks, or none. An agent that
// does not answer lists none, as while Secretive starts.
func (k *MacKey) fresh(ctx context.Context, before []sshcfg.AgentKey) sshcfg.AgentKey {
	keys, _, _ := setup.SecretiveKeys(ctx, k.M, k.Home)
	return unlisted(keys, before)
}

// unlisted returns the first key of keys that before lacks, or none.
func unlisted(keys, before []sshcfg.AgentKey) sshcfg.AgentKey {
	for _, key := range keys {
		if !slices.ContainsFunc(before, func(b sshcfg.AgentKey) bool { return b.Name == key.Name }) {
			return key
		}
	}
	return sshcfg.AgentKey{}
}

// answer returns the keys that Secretive's agent lists, once it answers, as it does once Secretive
// starts. It waits up to 15 minutes, and says so where the agent is not there yet.
func (k *MacKey) answer(ctx context.Context) ([]sshcfg.AgentKey, error) {
	for left := 450; ; left-- {
		keys, there, err := setup.SecretiveKeys(ctx, k.M, k.Home)
		switch {
		case there && err == nil:
			return keys, nil
		case ctx.Err() != nil:
			return nil, ctx.Err()
		case left < 0:
			return nil, fmt.Errorf("%s: Secretive's agent did not answer in 15 minutes: open Secretive, then run the "+
				"wizard again", macKey)
		case left == 450:
			k.Out.Say("%s: the wizard waits up to 15 minutes for Secretive's agent, which its app starts", macKey)
		}
		k.Sleep(2 * time.Second)
	}
}

// wait says to make what in Secretive's app, and waits up to 15 minutes for a key that the agent
// lists besides those of before. It fails on a key other than P-256, as an ML-DSA one, which ws's
// OpenSSH lacks.
func (k *MacKey) wait(ctx context.Context, before []sshcfg.AgentKey, what string) (sshcfg.AgentKey, error) {
	k.Out.Say("%s: in Secretive, make %s; the wizard waits up to 15 minutes for it", macKey, what)
	got := k.fresh(ctx, before)
	for left := 450; got.Name == ""; got = k.fresh(ctx, before) {
		// A signal ends the wait at once, as a trap did in sh.
		if err := ctx.Err(); err != nil {
			return got, err
		}
		if left--; left < 0 {
			return got, fmt.Errorf("%s: Secretive made no key in 15 minutes: make %s, then run the wizard again",
				macKey, what)
		}
		k.Sleep(2 * time.Second)
	}
	if !got.P256() {
		return got, fmt.Errorf("%s: %s is not the P-256 key asked for: remove it in Secretive, make %s, then run the "+
			"wizard again", macKey, got.Label(), what)
	}
	return got, nil
}
