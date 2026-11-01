package setup_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

const cldScript = "/tmp/ws-cld.1"

func cldApply(t *testing.T, check bool, fix func(*effecttest.Machine)) ([]string, []string) {
	s, m, out := cldStep(check)
	delete(m.Files, cldBin)
	m.Results["mktemp -t ws-cld"] = effecttest.Result{Out: []byte(cldScript + "\n")}
	m.Results["rm -f "+cldScript] = effecttest.Result{}
	m.Results["curl -fsSL --retry 3 --proto-redir =https -o "+cldScript+
		" https://github.com/zadykian/cld/releases/latest/download/install.sh"] = effecttest.Result{}
	m.Results["env CLD_INSTALL_DIR=/h/.local/bin sh "+cldScript] = effecttest.Result{}
	fix(m)
	p, err := s.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	again, _ := s.Plan(context.Background())
	if len(again.Changes) != 0 {
		t.Errorf("second Plan = %v; want none", again.Changes)
	}
	return m.Calls, out.Lines
}

func TestCldInstallRunsItsScriptAndRemovesIt(t *testing.T) {
	calls, lines := cldApply(t, false, func(*effecttest.Machine) {})
	got := strings.Join(calls, "\n")
	if !strings.Contains(got, "run env CLD_INSTALL_DIR=/h/.local/bin sh "+cldScript) ||
		!strings.HasSuffix(got, "run rm -f "+cldScript) || len(lines) != 0 {
		t.Errorf("calls %q, out %q; want the script run, then removed, and no warning", calls, lines)
	}
}

func TestCldInstallWarnsOfAScriptThatFails(t *testing.T) {
	_, lines := cldApply(t, false, func(m *effecttest.Machine) {
		m.Results["env CLD_INSTALL_DIR=/h/.local/bin sh "+cldScript] = effecttest.Result{Err: errors.New("no")}
	})
	if len(lines) != 1 || lines[0] != "warn: cld's install.sh failed" {
		t.Errorf("out = %q; want the warning of install.sh", lines)
	}
}

func TestCldUpdateSaysWhatCldPrints(t *testing.T) {
	s, m, out := cldStep(false)
	m.Results[`sh -c exec "$0" update 2>&1 `+cldBin] = effecttest.Result{Out: []byte("Updated cld 0.13.0 to 0.14.0\n")}
	p, _ := s.Plan(context.Background())
	if err := s.Apply(context.Background(), p); err != nil || len(out.Lines) != 1 ||
		out.Lines[0] != "say: Updated cld 0.13.0 to 0.14.0" {
		t.Errorf("Apply = %v, out %q; want the line cld printed", err, out.Lines)
	}
}
