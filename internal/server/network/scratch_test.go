package network

import (
	"context"
	"errors"
	"io/fs"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const scratch = "/tmp/x.netplan"

func netplanMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Dirs["/etc/netplan"] = 0o755
	m.Dirs["/lib/netplan"] = 0o755
	m.Results["mktemp -d --suffix=.netplan"] = effecttest.Result{Out: []byte(scratch + "\n")}
	m.Results["cp -a /etc/netplan/. "+scratch+"/etc/netplan/"] = effecttest.Result{}
	m.Results["cp -a /lib/netplan/. "+scratch+"/lib/netplan/"] = effecttest.Result{}
	m.Results["/usr/libexec/netplan/configure --root-dir "+scratch] = effecttest.Result{}
	m.Results["netplan get --root-dir "+scratch] = effecttest.Result{Out: []byte("network:\n  ethernets:\n    enp5s0: {}\n")}
	return m
}

func TestCheckNetplan(t *testing.T) {
	m := netplanMachine()
	problems, err := checkNetplan(context.Background(), m, []byte("file"), cable)
	if err != nil || len(problems) != 0 {
		t.Fatalf("problems = %q, err = %v", problems, err)
	}
	for _, want := range []string{
		"root cp -a /etc/netplan/. " + scratch + "/etc/netplan/",
		"root cp -a /lib/netplan/. " + scratch + "/lib/netplan/",
		"write root " + scratch + netplanPath + " 600",
		"root rm -rf -- " + scratch,
	} {
		if !slices.Contains(m.Calls, want) {
			t.Errorf("missing the call %q in %q", want, m.Calls)
		}
	}
	for _, call := range m.Calls {
		if strings.HasPrefix(call, "root cp -a /run/") {
			t.Errorf("copied a directory the machine lacks: %s", call)
		}
	}
	if got := string(m.Files[scratch+netplanPath]); got != "file" {
		t.Errorf("the scratch file = %q", got)
	}
}

func TestCheckNetplanWordsNetplansRefusal(t *testing.T) {
	m := netplanMachine()
	m.Results["/usr/libexec/netplan/configure --root-dir "+scratch] = effecttest.Result{
		Err: &exec.ExitError{Stderr: []byte("Command failed: bad key.\n  password: hunter2\n")},
	}
	problems, err := checkNetplan(context.Background(), m, nil, cable)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"netplan rejects its files with the direct link's: bad key."}; !slices.Equal(problems, want) {
		t.Errorf("problems = %q, want %q", problems, want)
	}
	if !slices.Contains(m.Calls, "root rm -rf -- "+scratch) {
		t.Error("the scratch root stays")
	}
}

func TestCheckNetplanFails(t *testing.T) {
	m := netplanMachine()
	m.Results["cp -a /etc/netplan/. "+scratch+"/etc/netplan/"] = effecttest.Result{Err: fs.ErrPermission}
	if _, err := checkNetplan(context.Background(), m, nil, cable); !errors.Is(err, fs.ErrPermission) {
		t.Errorf("err = %v, want a permission error", err)
	}
	if !slices.Contains(m.Calls, "root rm -rf -- "+scratch) {
		t.Error("the scratch root stays")
	}
}

func TestRunRootKeepsAFailureToStart(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["netplan get"] = effecttest.Result{Err: errors.New("exec: not found")}
	if got := runRoot(context.Background(), m, "netplan", "get"); got.RC == 0 || got.Msg != "exec: not found" {
		t.Errorf("result = %+v", got)
	}
}
