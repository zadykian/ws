package daemon

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/wgkey"
)

// keyMachine moves a file for mv, as fakeMachine does.
func keyMachine() *fakeMachine {
	return &fakeMachine{Machine: effecttest.NewMachine(), table: map[string]effecttest.Result{}}
}

func TestKeyMakesOne(t *testing.T) {
	m := keyMachine()
	random := bytes.NewReader(bytes.Repeat([]byte{7}, 32))
	public, err := Key(context.Background(), m, random, dir)
	if err != nil {
		t.Fatal(err)
	}
	private, err := ReadKey(m, dir)
	if err != nil {
		t.Fatalf("private.key: %v", err)
	}
	if want, _ := wgkey.Public(private); public != want {
		t.Errorf("public = %q, want %q", public, want)
	}
	if got := strings.Join(m.runs, ";"); got != "mv -f "+dir+"/private.key.new "+dir+"/private.key" {
		t.Errorf("commands = %q", got)
	}
}

func TestKeyKeepsTheOneThatIsThere(t *testing.T) {
	m := keyMachine()
	m.Files[dir+"/private.key"] = []byte(macKey + "\n")
	public, err := Key(context.Background(), m, strings.NewReader(""), dir)
	if want, _ := wgkey.Public(macKey); err != nil || public != want {
		t.Errorf("Key = %q, %v, want %q", public, err, want)
	}
	if len(m.runs) != 0 || string(m.Files[dir+"/private.key"]) != macKey+"\n" {
		t.Errorf("replaced the key: commands %q", m.runs)
	}
}

func TestKeyFails(t *testing.T) {
	bad := keyMachine()
	bad.Files[dir+"/private.key"] = []byte("nope\n")
	short := keyMachine()
	tests := []struct {
		name   string
		m      *fakeMachine
		random string
		want   string
	}{
		{"not a key", bad, "", "not a WireGuard key"},
		{"no random", short, "abc", "cannot make a key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Key(context.Background(), tt.m, strings.NewReader(tt.random), dir)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want one with %q", err, tt.want)
			}
		})
	}
	if _, ok := bad.Files[dir+"/private.key.new"]; ok {
		t.Error("wrote a new key beside one that is there")
	}
}
