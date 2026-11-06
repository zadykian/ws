package setup_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

// reportOf reads the step as ws mac state does, and returns its items by name.
func reportOf(t *testing.T, s step.Step) map[string]step.Item {
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

func TestSSHReportLeavesTheUsersConfigOut(t *testing.T) {
	m := sshMachine()
	m.Files["/h/.ssh/config"] = []byte("Host ws\n    IdentityFile ~/.ssh/hidden\n")
	it := reportOf(t, sshStep(m, &effecttest.Out{}))["/h/.ssh/config"]
	if it.Found != "lacks it" || !strings.Contains(it.Conflict, "line 1") || it.Diff != "" {
		t.Errorf("config = %+v, want the Include lacking, and line 1 named", it)
	}
	if strings.Contains(fmt.Sprintf("%+v", it), "hidden") {
		t.Errorf("config = %+v, which shows the user's file", it)
	}
}
