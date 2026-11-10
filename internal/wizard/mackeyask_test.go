package wizard

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
)

// The labels of p256 and gitP256, by their comments and fingerprints.
const (
	p256Label = "me (SHA256:cJ6AyISHokEeHuTfufIqhhSS0gxHZRUMDHlKvXD4FHw)"
	gitLabel  = "me (SHA256:CcCKY/wrEaUM+I629sBixyeWSgyCiAj+RsdArzoziXo)"
	gitConf   = "WS_GIT_NAME=ada\nWS_GIT_EMAIL=a@b.c\n"
)

// chooser answers each choice with answer, the default where empty, and keeps the questions.
type chooser struct {
	effect.Prompt
	answer string
	asked  []effect.ChoiceQuestion
}

func (c *chooser) Choose(_ context.Context, q effect.ChoiceQuestion) (string, error) {
	c.asked = append(c.asked, q)
	return cmp.Or(c.answer, q.Default), nil
}

func TestMacKeyQuestionString(t *testing.T) {
	want := "Mac key: would ask for git's signing key, in Secretive: " + p256Label + ", ws's login key; " +
		gitLabel + "; make a new one (the default)"
	q := keyQuestion(gitUse, listed(p256, mldsa, gitP256), listed(p256)[0].Name)
	if got := (askKey{gitUse, q}).String(); got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}

// One key may serve both: git's question marks the login key, which the answer names for git too.
func TestMacKeyQuestionForGit(t *testing.T) {
	login := listed(p256)[0].Pub()
	k, m, _ := newMacKey(t, gitConf+"WS_IDENTITY_PUB_KEY="+login+"\n")
	agent(m, p256, gitP256)
	c := &chooser{answer: p256Label + ", ws's login key"}
	k.Tty, k.P = true, c
	if err := k.ask(context.Background(), gitUse); err != nil {
		t.Fatal(err)
	}
	if want := []string{p256Label + ", ws's login key", gitLabel, newKey}; !slices.Equal(c.asked[0].Options, want) {
		t.Errorf("asked %+v, want the options %q", c.asked, want)
	}
	if got := string(m.Files[confPathForTest]); got != gitConf+"WS_IDENTITY_PUB_KEY="+login+"\nWS_GIT_SIGNING_PUB_KEY="+login+"\n" {
		t.Errorf("mac.conf = %q, want git's line added", got)
	}
}

func TestMacKeyQuestionMakesANewKey(t *testing.T) {
	k, m, out := newMacKey(t, gitConf+"WS_GIT_SIGNING_PUB_KEY=\n")
	agent(m, p256)
	m.Dirs["/h/Applications/Secretive.app"] = 0o755
	m.Results["open -a Secretive"] = effecttest.Result{}
	k.Tty, k.P = true, &chooser{}
	k.Sleep = func(time.Duration) { agent(m, p256, gitP256) }
	if err := k.ask(context.Background(), gitUse); err != nil {
		t.Fatalf("ask = %v, saying %q", err, out.Lines)
	}
	if got := string(m.Files[confPathForTest]); got != gitConf+"WS_GIT_SIGNING_PUB_KEY="+listed(gitP256)[0].Pub()+"\n" {
		t.Errorf("mac.conf = %q, want the new key's line in place of the empty one", got)
	}
}

func TestMacKeyQuestionWithNoTerminal(t *testing.T) {
	want := "Mac key: no terminal to ask for git's signing key: add one of these lines to mac.conf, " +
		"WS_GIT_SIGNING_PUB_KEY=" + listed(p256)[0].Pub() + " for " + p256Label + "; or run the wizard at a terminal"
	if err := noTerminal(gitUse, listed(p256, mldsa)); err == nil || err.Error() != want {
		t.Errorf("noTerminal = %v, want %q", err, want)
	}
	if err := noTerminal(gitUse, listed(mldsa)); err == nil || !strings.Contains(err.Error(), "lists no P-256 key") {
		t.Errorf("noTerminal of no P-256 key = %v", err)
	}
}

// name writes ws's copy of the key's .pub, and its line in place of an empty one.
func TestMacKeyName(t *testing.T) {
	k, m, _ := newMacKey(t, gitConf+"WS_GIT_SIGNING_PUB_KEY=\n")
	agent(m, gitP256)
	if err := k.name(context.Background(), gitUse, listed(gitP256)[0]); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[confPathForTest]); got != gitConf+"WS_GIT_SIGNING_PUB_KEY="+listed(gitP256)[0].Pub()+"\n" {
		t.Errorf("mac.conf = %q", got)
	}
	if string(m.Files[copyOf(gitP256)]) != gitP256 {
		t.Errorf("the copy holds %q", m.Files[copyOf(gitP256)])
	}
}
