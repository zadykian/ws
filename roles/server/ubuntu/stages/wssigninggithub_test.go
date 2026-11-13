package stages

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const loginCmd = "runuser -l ada -c gh api user --jq .login 2>/dev/null || git config --global github.user"

func TestGithubLoginOf(t *testing.T) {
	for out, want := range map[string]string{"motd\nada-1\n": "ada-1", "a/b\n": "", "": "", "-x\n": ""} {
		m := effecttest.NewMachine()
		m.Results[loginCmd] = effecttest.Result{Out: []byte(out)}
		if got := githubLoginOf(context.Background(), m, "ada"); got != want {
			t.Errorf("login of %q = %q, want %q", out, got, want)
		}
	}
	m := effecttest.NewMachine()
	m.Results[loginCmd] = effecttest.Result{Out: []byte("ada\n"), Err: errors.New("exit status 1")}
	if got := githubLoginOf(context.Background(), m, "ada"); got != "" {
		t.Errorf("login where runuser fails = %q", got)
	}
}

func TestSigningKeysOf(t *testing.T) {
	m := effecttest.NewMachine()
	url := "https://api.github.com/users/ada-1/ssh_signing_keys?per_page=100"
	m.Results["curl -fsSL --retry 3 --proto =https "+url] = effecttest.Result{
		Out: []byte(`[{"id":1,"key":"ssh-ed25519 AAA"},{"key":"ssh-rsa BBB c"},{"key":""}]`)}
	keys, err := signingKeysOf(context.Background(), m, "ada-1")
	if err != nil || !slices.Equal(keys, []string{"ssh-ed25519 AAA", "ssh-rsa BBB"}) {
		t.Errorf("keys = %q, %v", keys, err)
	}
	m.Results["curl -fsSL --retry 3 --proto =https "+url] = effecttest.Result{Out: []byte("<html>")}
	if _, err := signingKeysOf(context.Background(), m, "ada-1"); err == nil {
		t.Error("a page that isn't JSON passed")
	}
}
