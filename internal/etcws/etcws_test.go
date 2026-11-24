package etcws_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/etcws"
)

func TestFilesMakeTheDirectoryBeforeTheFile(t *testing.T) {
	m := effecttest.NewMachine()
	var got []string
	for _, r := range etcws.Files(m, etcws.Dir+"/x", []byte("a\n")) {
		changes, err := r.Plan(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range changes {
			got = append(got, strings.SplitN(c.String(), "\n", 2)[0])
		}
	}
	want := []string{"create directory /etc/ws, owner 0:0, mode 755", "create /etc/ws/x"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("changes = %q, want %q", got, want)
	}
}

func TestTopIsTheDirectoryAlone(t *testing.T) {
	m := effecttest.NewMachine()
	changes, err := etcws.Top(m).Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || !strings.HasPrefix(changes[0].String(), "create directory /etc/ws, owner 0:0, mode 755") {
		t.Errorf("changes = %q, want one that makes /etc/ws at 755", changes)
	}
}

func TestFilesMakeEachDirectoryBetween(t *testing.T) {
	m := effecttest.NewMachine()
	var got []string
	for _, r := range etcws.Files(m, etcws.Dir+"/servers/lab/values.yml", []byte("a\n")) {
		changes, err := r.Plan(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range changes {
			got = append(got, strings.SplitN(c.String(), "\n", 2)[0])
		}
	}
	want := []string{
		"create directory /etc/ws, owner 0:0, mode 755",
		"create directory /etc/ws/servers, owner 0:0, mode 755",
		"create directory /etc/ws/servers/lab, owner 0:0, mode 755",
		"create /etc/ws/servers/lab/values.yml",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("changes = %q, want %q", got, want)
	}
}

func TestInsideLeavesOutTheTop(t *testing.T) {
	m := effecttest.NewMachine()
	var got []string
	for _, r := range etcws.Inside(m, etcws.Dir+"/secretive/a.pub", []byte("a\n")) {
		changes, err := r.Plan(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range changes {
			got = append(got, strings.SplitN(c.String(), "\n", 2)[0])
		}
	}
	want := []string{"create directory /etc/ws/secretive, owner 0:0, mode 755", "create /etc/ws/secretive/a.pub"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("changes = %q, want %q", got, want)
	}
}

func TestHolds(t *testing.T) {
	for p, want := range map[string]bool{
		"/etc/ws/x": true, "/etc/ws/servers/a/values.yml": true,
		"/etc/ws/../x": false, "/etc/wsx": false, "/etc/ws": false, "/h/x": false,
	} {
		if got := etcws.Holds(p); got != want {
			t.Errorf("Holds(%q) = %v, want %v", p, got, want)
		}
	}
}

func rootCalls(m *effecttest.Machine) []string {
	var got []string
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "root ") {
			got = append(got, c)
		}
	}
	return got
}

func TestWriteMakesTheDirectoriesAndTheFile(t *testing.T) {
	m := effecttest.NewMachine()
	var out effecttest.Out
	path := etcws.Dir + "/servers/lab/values.yml"
	if err := etcws.Write(context.Background(), m, &out, path, []byte("a\n")); err != nil {
		t.Fatal(err)
	}
	if string(m.Files[path]) != "a\n" || m.Modes[path] != etcws.FileMode || m.Owners[path] != [2]int{} {
		t.Errorf("file = %q, mode %o, owner %v; want root's at 644", m.Files[path], m.Modes[path], m.Owners[path])
	}
	for _, d := range []string{etcws.Dir, etcws.Dir + "/servers", etcws.Dir + "/servers/lab"} {
		if mode, ok := m.Dirs[d]; !ok || mode != etcws.DirMode || m.Owners[d] != [2]int{} {
			t.Errorf("%s = %o, %v; want root's at 755", d, mode, ok)
		}
	}
	if len(out.Lines) != 4 {
		t.Errorf("said %q, want one line for each of four changes", out.Lines)
	}
}

func TestWriteAgainRunsNoSudo(t *testing.T) {
	m := effecttest.NewMachine()
	path := etcws.Dir + "/values.yml"
	if err := etcws.Write(context.Background(), m, &effecttest.Out{}, path, []byte("a\n")); err != nil {
		t.Fatal(err)
	}
	before := len(rootCalls(m))
	var out effecttest.Out
	if err := etcws.Write(context.Background(), m, &out, path, []byte("a\n")); err != nil {
		t.Fatal(err)
	}
	if got := rootCalls(m); len(got) != before || len(out.Lines) != 0 {
		t.Errorf("second write: %d root calls (was %d), said %q; want none", len(got), before, out.Lines)
	}
}

func TestWriteKeepsAPrivateDirectoryClosed(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs[etcws.Dir] = etcws.DirMode
	m.Dirs[etcws.Dir+"/tunnel"] = 0o700
	var out effecttest.Out
	err := etcws.Write(context.Background(), m, &out, etcws.Dir+"/tunnel/values.yml", []byte("a\n"))
	if err == nil || !strings.Contains(err.Error(), etcws.Dir+"/tunnel") {
		t.Fatalf("err = %v, want one that names /etc/ws/tunnel", err)
	}
	if mode := m.Dirs[etcws.Dir+"/tunnel"]; mode != 0o700 || len(m.Files) != 0 || len(out.Lines) != 0 || len(rootCalls(m)) != 0 {
		t.Errorf("mode %o, files %v, said %q, root calls %v; want nothing changed", mode, m.Files, out.Lines, rootCalls(m))
	}
}
