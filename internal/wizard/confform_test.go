package wizard

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/form"
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
func keysOf(fields []form.Field) []string {
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

// fieldOf finds the key's field.
func fieldOf(t *testing.T, fields []form.Field, key string) form.Field {
	t.Helper()
	at := slices.IndexFunc(fields, func(f form.Field) bool { return f.Key == key })
	if at < 0 {
		t.Fatalf("no field of %s in %v", key, keysOf(fields))
	}
	return fields[at]
}

func TestConfFormEditLeadsTheWalk(t *testing.T) {
	f, _ := formOf(t, "")
	fields := f.Fields(map[string]string{"WS_REMOTE": "ws.example", "WS_GIT_NAME": "Me"})
	if got := fieldOf(t, fields, "WS_REMOTE_PORT").Text; got != "The router's port forwarded to ws's 22 (the wizard offers 22022)" {
		t.Errorf("port's text = %q", got)
	}
	if got := fieldOf(t, fields, "WS_GIT_EMAIL").Default; got != "a@b.dev" {
		t.Errorf("email's default = %q", got)
	}
	if got := fieldOf(t, f.Fields(nil), "WS_TARGET").Options; !slices.Equal(got, targets) {
		t.Errorf("target's options = %v", got)
	}
}

func TestConfFormFieldsLeaveTheFile(t *testing.T) {
	f, m := formOf(t, "WS_REMOTE=r.example\n")
	f.Fields(map[string]string{"WS_REMOTE": "other.example"})
	if got := string(m.Files["/h/mac.conf"]); got != "WS_REMOTE=r.example\n" {
		t.Errorf("mac.conf = %q", got)
	}
}

func TestConfFormValidReadsTheForm(t *testing.T) {
	f, _ := formOf(t, "WS_TARGET=cloud\nWS_REMOTE=vm.example\nWS_REMOTE_PORT=22\n")
	remote := fieldOf(t, f.Fields(nil), "WS_REMOTE")
	if err := remote.Valid(""); err == nil || err.Error() != "WS_TARGET=cloud needs WS_REMOTE and WS_REMOTE_PORT" {
		t.Errorf("empty name: %v", err)
	}
	if err := fieldOf(t, f.Fields(nil), "WS_TUNNEL_NETWORKS").Valid(""); err != nil {
		t.Errorf("empty networks: %v", err)
	}
	address := fieldOf(t, f.Fields(map[string]string{"WS_TARGET": "lan", "WS_LAN": "10.88.0.0/24"}), "WS_LAN_ADDRESS")
	if err := address.Valid("10.9.0.1"); err == nil {
		t.Error("an address outside the LAN passes")
	}
}

func TestConfFormSave(t *testing.T) {
	way := "WS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.10\nWS_REMOTE=r.example\nWS_REMOTE_PORT=22022\n"
	for name, tc := range map[string]struct {
		text  string
		edits map[string]string
		want  string
	}{
		"a needed default": {"", map[string]string{"WS_TARGET": "cloud", "WS_REMOTE": "vm.example"},
			"WS_TARGET=cloud\nWS_REMOTE=vm.example\nWS_REMOTE_PORT=22\n"},
		"no optional default": {"", map[string]string{"WS_GIT_NAME": "A B"},
			"WS_GIT_NAME=\"A B\"\nWS_GIT_EMAIL=a@b.dev\n"},
		"to a cloud VM": {way + "WS_CABLE_PORT=enp1s0\n", map[string]string{"WS_TARGET": "cloud"},
			"WS_LAN=\nWS_LAN_ADDRESS=\nWS_REMOTE=r.example\nWS_REMOTE_PORT=22022\nWS_CABLE_PORT=\nWS_TARGET=cloud\n"},
		"a cleared field": {way, map[string]string{"WS_REMOTE": ""},
			"WS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.10\nWS_REMOTE=\nWS_REMOTE_PORT=\n"},
		"none and quotes": {"WS_TUNNEL_NETWORKS=10.20.0.0/16\n",
			map[string]string{"WS_TUNNEL_NETWORKS": "none", "WS_GIT_NAME": `"Me"`, "WS_GIT_EMAIL": "me@b.dev"},
			"WS_TUNNEL_NETWORKS=\nWS_GIT_NAME=Me\nWS_GIT_EMAIL=me@b.dev\n"},
		"a mended pair": {"WS_LAN=10.88.0.0/24\n", nil, "WS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.10\n"},
	} {
		f, m := formOf(t, tc.text)
		if _, err := f.Save(context.Background(), &effecttest.Out{}, tc.edits); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		if got := string(m.Files["/h/mac.conf"]); got != tc.want {
			t.Errorf("%s: mac.conf = %q; want %q", name, got, tc.want)
		}
	}
}

func TestConfFormSaveReturnsWhatTheFileHolds(t *testing.T) {
	f, _ := formOf(t, "")
	next, err := f.Save(context.Background(), &effecttest.Out{}, map[string]string{"WS_REMOTE": "r.example"})
	if err != nil {
		t.Fatal(err)
	}
	if got := fieldOf(t, next.Fields(nil), "WS_REMOTE").Value; got != "r.example" {
		t.Errorf("value = %q", got)
	}
	if got := fieldOf(t, f.Fields(nil), "WS_REMOTE").Value; got != "" {
		t.Errorf("the form before the save holds %q", got)
	}
}

func TestConfFormSaveOfNoChangeWritesNothing(t *testing.T) {
	f, m := formOf(t, "# mine\nWS_REMOTE=r.example\n")
	out := &effecttest.Out{}
	if _, err := f.Save(context.Background(), out, nil); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/h/mac.conf"]); got != "# mine\nWS_REMOTE=r.example\n" {
		t.Errorf("mac.conf = %q", got)
	}
	if want := []string{"say: /h/mac.conf holds these values already"}; !slices.Equal(out.Lines, want) {
		t.Errorf("said %q; want %q", out.Lines, want)
	}
}

func TestConfFormSaveSaysWhatItEmptied(t *testing.T) {
	f, _ := formOf(t, "WS_REMOTE=r.example\nWS_REMOTE_PORT=22022\n")
	out := &effecttest.Out{}
	if _, err := f.Save(context.Background(), out, map[string]string{"WS_REMOTE": ""}); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(out.Lines, "say: emptied WS_REMOTE WS_REMOTE_PORT") {
		t.Errorf("said %q", out.Lines)
	}
}

func TestConfFormSaveRefusals(t *testing.T) {
	signed := "WS_GIT_NAME=Me\nWS_GIT_EMAIL=me@b.dev\nWS_GIT_SIGNING_PUB_KEY=~/.ssh/k.pub\n"
	f, m := formOf(t, signed)
	_, err := f.Save(context.Background(), &effecttest.Out{}, map[string]string{"WS_GIT_NAME": ""})
	if !errors.Is(err, errRefused) || string(m.Files["/h/mac.conf"]) != signed {
		t.Errorf("a signing key with no name: %v, mac.conf = %q", err, m.Files["/h/mac.conf"])
	}
	f, m = formOf(t, "")
	m.Links["/h/mac.conf"] = "/dotfiles/mac.conf"
	out := &effecttest.Out{}
	if _, err := f.Save(context.Background(), out, map[string]string{"WS_REMOTE": "r.example"}); !errors.Is(err, errRefused) {
		t.Errorf("a link: %v", err)
	}
	if said := strings.Join(out.Lines, "\n"); !strings.Contains(said, "is a link") || !strings.Contains(said, "WS_REMOTE=r.example") {
		t.Errorf("said %q", said)
	}
}
