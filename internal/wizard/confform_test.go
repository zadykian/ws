package wizard

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// formOf returns the form of mac.conf's text.
func formOf(t *testing.T, text string) (ConfForm, *effecttest.Machine) {
	t.Helper()
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte(text)
	m.Results["mkdir -p /h"] = effecttest.Result{}
	f, err := ConfForm{Conf: "/h/mac.conf", Defaults: []byte("shell_git_name: A B\nshell_git_email: a@b.dev\n"), M: m}.
		Load(&effecttest.Out{})
	if err != nil {
		t.Fatal(err)
	}
	return f, m
}

func TestConfFormLoadReadsTheFile(t *testing.T) {
	f, _ := formOf(t, "WS_TARGET=cloud\n")
	if got := f.file.Target; got != "cloud" {
		t.Errorf("target = %q", got)
	}
}

func TestConfFormLoadRefusesABadLine(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte("no line\n")
	if _, err := (ConfForm{Conf: "/h/mac.conf", M: m}).Load(&effecttest.Out{}); err == nil {
		t.Error("Load = nil; want a refusal")
	}
}

// keysOf lists the keys that show.
func keysOf(fields []Field) []string {
	keys := make([]string, 0, len(fields))
	for _, f := range fields {
		keys = append(keys, f.Key)
	}
	return keys
}

func TestConfFormShowsTheKeysOfATarget(t *testing.T) {
	tail := []string{"WS_GIT_NAME", "WS_TUNNEL_NETWORKS", "WS_RESOLVERS"}
	for target, want := range map[string][]string{
		"lan":    slices.Concat([]string{"WS_TARGET", "WS_LAN", "WS_LAN_ADDRESS", "WS_REMOTE"}, tail),
		"cloud":  slices.Concat([]string{"WS_TARGET", "WS_REMOTE", "WS_REMOTE_PORT"}, tail),
		"mac-vm": slices.Concat([]string{"WS_TARGET", "WS_LAN", "WS_LAN_ADDRESS"}, tail),
	} {
		f, _ := formOf(t, "WS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.10\n")
		if got := keysOf(f.Fields(map[string]string{"WS_TARGET": target})); !slices.Equal(got, want) {
			t.Errorf("%s: keys = %v; want %v", target, got, want)
		}
	}
}
