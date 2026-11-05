package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

type textChange string

func (c textChange) String() string { return string(c) }

type kindedChange struct{ textChange }

func (kindedChange) Kind() step.ChangeKind { return step.Install }

type recorder []step.Event

func (r *recorder) Observe(e step.Event) { *r = append(*r, e) }

func TestJSONEventsPrintALinePerEventAndPassItOn(t *testing.T) {
	var b bytes.Buffer
	var next recorder
	j := newJSONEvents(&b, &next)
	events := []step.Event{
		{Kind: step.Planned, Parent: "base", Step: "Packages", Changes: []step.Change{kindedChange{"install git"}, textChange("note")}},
		{Kind: step.Started, Parent: "base", Step: "Packages", Index: 1},
		{Kind: step.Failed, Parent: "base", Step: "Packages", Index: 1, Err: errors.New("boom")},
		{Kind: step.Skipped, Parent: "base", Step: "Later", Index: 2, After: "Packages", Check: true},
	}
	for _, e := range events {
		j.Observe(e)
	}
	got := b.String()
	if lines := strings.Count(got, "\n"); lines != len(events) {
		t.Fatalf("got %d lines, want %d:\n%s", lines, len(events), got)
	}
	for _, frag := range []string{`"event":"failed","parent":"base","step":"Packages","index":1,"error":"boom"}`,
		`"event":"skipped","parent":"base","step":"Later","index":2,"check":true,"after":"Packages"}`,
		`"text":"install git"`, `"text":"note"`} {
		if !strings.Contains(got, frag) {
			t.Errorf("output lacks %s:\n%s", frag, got)
		}
	}
	if len(next) != len(events) {
		t.Errorf("next saw %d events, want %d", len(next), len(events))
	}
	if j.err != nil {
		t.Errorf("err = %v", j.err)
	}
}

func TestServerApplyHelpNamesJSON(t *testing.T) {
	status, stdout, _ := runWs(fakeProcess(), "", "server", "apply", "-h")
	if status != 0 || !strings.Contains(stdout, "--json") || !strings.Contains(stdout, "no screen is drawn") {
		t.Errorf("status = %d, help = %q", status, stdout)
	}
}

func TestServerApplyJSONRefusesNonRootOnStdoutAlone(t *testing.T) {
	p := fakeProcess()
	p.euid = 1000
	status, stdout, stderr := runWs(p, "", "server", "apply", "--json")
	if status != 1 || stdout != "" || !strings.Contains(stderr, "run it as root") {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

// A terminal changes nothing for --json: the events reach stdout, and no screen takes them.
func TestServerApplyJSONDrawsNoScreenAtATerminal(t *testing.T) {
	p, m := serverTerminal()
	m.Results["dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n tmux"] = effecttest.Result{Out: []byte("tmux ii \n")}
	status, stdout, stderr := runWs(p, "", "server", "apply", "--json", "--check", "--only", "tmux")
	if status != 0 || !strings.Contains(stdout, `"event":"done","parent":"tmux"`) || strings.Contains(stderr, enterScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

// With no selection the list would open here. The run ends in Prepare, so that no role runs.
func TestServerApplyJSONOpensNoListAtATerminal(t *testing.T) {
	p, m := serverTerminal()
	m.Results[dpkgCheck] = effecttest.Result{Err: errors.New("no dpkg")}
	status, stdout, stderr := runWs(p, "q", "server", "apply", "--json")
	if status != 1 || !strings.Contains(stdout, `"event":"failed"`) || strings.Contains(stderr, enterScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}
