package shell

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

func stepNames(v values.Values, s secrets.Secrets) []string {
	var names []string
	for _, st := range Steps(v, s, effecttest.NewMachine(), &effecttest.Units{}, &effecttest.Out{}) {
		names = append(names, st.Name())
	}
	return names
}

func TestStepsAreTheRolesInOrder(t *testing.T) {
	got := stepNames(values.Default(), secrets.Secrets{})
	if want := []string{"Shell packages", "Shell files", keyStep, "Shell allowed signers", "Shell git config", "Shell secret env", "Shell git tokens", "Shell clones"}; !slices.Equal(got, want) {
		t.Errorf("steps = %v, want %v", got, want)
	}
	v := values.Default()
	v.Shell.GitSigning = false
	if got := stepNames(v, secrets.Secrets{}); slices.Contains(got, keyStep) {
		t.Errorf("steps without signing = %v, want no key", got)
	}
}

func TestALinkKeepsAReportFromThePathsBehindIt(t *testing.T) {
	v := values.Default()
	v.User, v.Home = "ws", "/home/ws"
	m := effecttest.NewMachine()
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Packages = effecttest.Packages{}
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Links["/home/ws/.bashrc"] = "/etc/shadow"
	m.Files["/home/ws/.bashrc"] = []byte("root:$6$hash:19000\n")
	reports, err := step.Read(t.Context(), Steps(v, secrets.Secrets{}, m, &effecttest.Units{}, &effecttest.Out{}))
	if err != nil {
		t.Fatal(err)
	}
	if text := fmt.Sprintf("%+v", reports); strings.Contains(text, "$6$hash") {
		t.Errorf("reports = %s, want nothing read through the link", text)
	}
	for _, r := range reports {
		if r.Step == "Shell files" {
			if len(r.Items) != 1 || !strings.Contains(r.Items[0].Conflict, "/etc/shadow") {
				t.Errorf("items = %+v, want the refusal", r.Items)
			}
			return
		}
	}
	t.Error("no report of Shell files")
}

func TestALinkAtAFileStopsItsStepAlone(t *testing.T) {
	v := values.Default()
	v.User, v.Home = "ws", "/home/ws"
	v.GitSigningKey = "/home/ws/.ssh/id_signing"
	m := effecttest.NewMachine()
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Packages = effecttest.Packages{}
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Links["/home/ws/.bashrc"] = "/srv/target"
	var failed []string
	for _, st := range Steps(v, secrets.Secrets{}, m, &effecttest.Units{}, &effecttest.Out{}) {
		if st.WaitsOn() != "" {
			continue // it waits on a step that runs apart from it here
		}
		if _, err := step.Run(context.Background(), []step.Step{st}, &effecttest.Out{}, true); err != nil {
			failed = append(failed, st.Name())
		}
	}
	if want := []string{"Shell files"}; !slices.Equal(failed, want) {
		t.Errorf("failed steps = %v, want %v", failed, want)
	}
}

func TestShellFilesTakeDpkgsArchitecture(t *testing.T) {
	v := values.Default()
	v.User, v.Home = "ws", "/home/ws"
	m := effecttest.NewMachine()
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("arm64\n")}
	steps := Steps(v, secrets.Secrets{}, m, &effecttest.Units{}, &effecttest.Out{})
	if _, err := step.Run(t.Context(), []step.Step{steps[1]}, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/home/ws/.bashrc"]); !strings.Contains(got, "DOTNET_RUNTIME_ID=linux-arm64") {
		t.Errorf(".bashrc = %q, want the arm64 runtime ID", got)
	}
}
