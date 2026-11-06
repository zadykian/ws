package wizard

import (
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
