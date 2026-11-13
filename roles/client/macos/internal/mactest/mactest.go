// Package mactest holds what the tests of the Mac's roles share. These are the machine with
// Secretive's agent and the report of a step, read as ws client macos state reads it.
package mactest

import (
	"context"
	"io/fs"
	"os/exec"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// The home that the agent's machine uses, Secretive's data and socket in it, the command that lists
// the agent's keys, and two keys.
const (
	AgentHome = "/h"
	AgentData = AgentHome + "/" + sshcfg.SecretiveData
	AgentSock = AgentData + "/socket.ssh"
	AgentList = "env SSH_AUTH_SOCK=" + AgentSock + " ssh-add -L"
	AgentKey  = "ecdsa-sha2-nistp256 AAAA me\n"
	AgentGit  = "ecdsa-sha2-nistp256 BBBB me\n"
)

// AgentMachine has Secretive's socket, whose agent lists the lines.
func AgentMachine(m *effecttest.Machine, lines ...string) *effecttest.Machine {
	m.Files[AgentSock], m.Modes[AgentSock] = nil, fs.ModeSocket
	m.Results[AgentList] = effecttest.Result{Out: []byte(strings.Join(lines, ""))}
	return m
}

// CopyOf is the path of ws's copy of the line's key.
func CopyOf(line string) string { return sshcfg.Expand(sshcfg.AgentKeys(line)[0].Pub(), AgentHome) }

// ReportOf reads the step as ws client macos state does, and returns its items by name.
func ReportOf(t *testing.T, s step.Step) map[string]step.Item {
	reports, err := step.Read(context.Background(), []step.Step{s})
	if err != nil {
		t.Fatal(err)
	}
	items := map[string]step.Item{}
	for _, it := range reports[0].Items {
		if it.Kind == step.CheckItem {
			t.Fatalf("the step failed: %s", it.Conflict)
		}
		items[it.Name] = it
	}
	return items
}

// ExitOne is the error of a command that exits 1.
func ExitOne(t *testing.T) error {
	err := exec.Command("sh", "-c", "exit 1").Run()
	if err == nil {
		t.Fatal("sh -c 'exit 1' succeeded")
	}
	return err
}

// UsersToken is a secret in the user's file, which a report never quotes.
const UsersToken = "FAKETOKEN0123"

// UsersText opens on a secret, as the lines beside ws's at a file's top are what a diff quotes.
const UsersText = "[url \"https://oauth2:" + UsersToken + "@example.test/\"]\n    insteadOf = https://example.test/\n"

// CheckUsersFile plans the step over the user's file that holds UsersText. The report and the log
// show ws's line alone of the file. --check's diff shows the lines beside it too.
func CheckUsersFile(t *testing.T, s step.Step, line string) {
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var shown, diffs strings.Builder
	for _, c := range p.Changes {
		shown.WriteString(step.PendingOf(c).Text + step.ExcerptOf(c))
		diffs.WriteString(c.String())
	}
	if strings.Contains(shown.String(), UsersToken) || !strings.Contains(shown.String(), line) {
		t.Errorf("report = %q, want %q and none of the user's text", shown.String(), line)
	}
	if !strings.Contains(diffs.String(), UsersToken) {
		t.Errorf("--check's diff lost its context: %q", diffs.String())
	}
}
