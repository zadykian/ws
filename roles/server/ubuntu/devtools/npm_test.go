package devtools

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
)

const wsDir = "/home/ws/.local/lib/language-servers"

// npmCommand is the command line that the fake machine answers for the user's npm ci.
func npmCommand(user, home string) string {
	switch user {
	case "root":
		return "env HOME=" + home + " " + npmBin + " ci --prefix " + serversDir(home) +
			" --ignore-scripts --no-audit --no-fund"
	default:
		return "runuser -u " + user + " -- env -C " + home + " HOME=" + home + " " + npmBin + " ci --prefix " +
			serversDir(home) + " --ignore-scripts --no-audit --no-fund"
	}
}

func newInstall(m *effecttest.Machine) npmInstall {
	return npmInstall{M: m, Account: testAccount(1000), Digest: "abc"}
}

func TestNpmPlansAChangeForAMissingNpmOrStamp(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{"no npm", map[string]string{}, 1},
		{"no stamp", map[string]string{npmBin: "x"}, 1},
		{"other digest", map[string]string{npmBin: "x", wsDir + "/node_modules/.ws-sha256": "old\n"}, 1},
		{"same digest", map[string]string{npmBin: "x", wsDir + "/node_modules/.ws-sha256": "abc\n"}, 0},
	}
	for _, c := range cases {
		m := effecttest.NewMachine()
		for p, body := range c.files {
			m.Files[p] = []byte(body)
		}
		got, err := newInstall(m).Plan(context.Background())
		if err != nil || len(got) != c.want {
			t.Errorf("%s: %d changes, %v; want %d", c.name, len(got), err, c.want)
		}
	}
}

func TestNpmCiRunsAsTheUserThenWritesTheStamp(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[npmBin] = []byte("x")
	m.Results[npmCommand("ws", "/home/ws")] = effecttest.Result{}
	n := newInstall(m)
	changes, err := n.Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("%v, %v", changes, err)
	}
	if err := changes[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if string(m.Files[n.stamp()]) != "abc\n" || m.Owners[n.stamp()] != [2]int{1000, 1000} {
		t.Errorf("stamp %q, owner %v", m.Files[n.stamp()], m.Owners[n.stamp()])
	}
	if !slices.Contains(m.Calls, "atomic "+n.stamp()+" 644") {
		t.Errorf("no stamp write in %v", m.Calls)
	}
}

func TestAFailedNpmCiLeavesNoStamp(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[npmBin] = []byte("x")
	m.Results[npmCommand("ws", "/home/ws")] = effecttest.Result{Err: context.Canceled}
	n := newInstall(m)
	changes, _ := n.Plan(context.Background())
	if err := changes[0].Apply(context.Background()); err == nil || len(m.Files[n.stamp()]) != 0 {
		t.Errorf("err %v, stamp %q", err, m.Files[n.stamp()])
	}
}

func TestNpmApplyNamesNodeWhereNpmIsMissing(t *testing.T) {
	m := effecttest.NewMachine()
	changes, _ := newInstall(m).Plan(context.Background())
	err := changes[0].Apply(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Node package") {
		t.Errorf("err %v", err)
	}
}

func TestTheStampIsTheDigestOfBothFiles(t *testing.T) {
	m := effecttest.NewMachine()
	parts, err := serverFiles(m, testAccount(1))
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	for _, name := range []string{"package.json", "package-lock.json"} {
		b, _ := serversLock.ReadFile("servers/" + name)
		h.Write(b)
	}
	if got := parts[len(parts)-2].(npmInstall).Digest; got != fmt.Sprintf("%x", h.Sum(nil)) {
		t.Errorf("digest %s", got)
	}
}

func TestTheFormatterLinkPlansBeforeItsTargetExists(t *testing.T) {
	m := effecttest.NewMachine()
	parts, err := serverFiles(m, testAccount(1000))
	if err != nil {
		t.Fatal(err)
	}
	link := parts[len(parts)-1]
	changes, err := link.Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("%v, %v", changes, err)
	}
	if err := changes[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if m.Links[formatterLink] != wsDir+"/node_modules/.bin/sql-formatter" {
		t.Errorf("link %q", m.Links[formatterLink])
	}
	if again, err := link.Plan(context.Background()); err != nil || len(again) != 0 {
		t.Errorf("a dangling link plans %v, %v", again, err)
	}
}

func TestTheServerFilesRefuseALink(t *testing.T) {
	m := effecttest.NewMachine()
	m.Links[wsDir] = "/srv/target"
	parts, err := serverFiles(m, testAccount(1000))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parts[0].Plan(context.Background()); err == nil {
		t.Error("a link at the servers' directory: no error")
	}
}

func TestTheStampIsNotReadThroughALink(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[npmBin] = []byte("x")
	m.Links[wsDir+"/node_modules"] = "/srv/target"
	if _, err := newInstall(m).Plan(context.Background()); err == nil {
		t.Error("a link at node_modules: no error")
	}
}

func testAccount(id int) host.Account {
	return host.Account{User: host.User{Name: "ws", Home: "/home/ws"}, UID: id, GID: id}
}
