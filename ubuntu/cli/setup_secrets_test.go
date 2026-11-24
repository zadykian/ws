package cli

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const (
	taskLine     = "/usr/local/bin/mise -C /opt/ws/ubuntu/guest run secrets"
	taskPlanLine = taskLine + " -- plan --detailed-exitcode"
)

func ran(m *effecttest.Machine, line string) bool {
	return slices.Contains(m.Calls, "run "+line)
}

func TestSetupSecretsRunsTheTaskInPlaceOfTheBootstrapAfterTheSteps(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	m.Results[taskLine] = effecttest.Result{}
	if status, stderr := runSetup(p, rel, "--secrets"); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	if !ran(m, taskLine) || ran(m, bootstrapLine) {
		t.Errorf("calls = %q", m.Calls)
	}
	if task, steps := indexOf(m, "run "+taskLine), indexOf(m, "run git -C /opt/ws clean"); task < steps {
		t.Errorf("the task ran at %d, before the tree's checkout at %d", task, steps)
	}
}

func TestSetupRunsTheTaskWhereTheClonesStandsAndTheBootstrapWhereItDoesNot(t *testing.T) {
	for name, clone := range map[string]bool{"a clone": true, "no clone": false} {
		t.Run(name, func(t *testing.T) {
			rel := setupRelease(miseBody)
			p, m := setupProcess(rel)
			m.Results[taskLine] = effecttest.Result{}
			if clone {
				m.Dirs[cloneDir] = 0o700
			}
			if status, stderr := runSetup(p, rel); status != 0 {
				t.Fatalf("status = %d, stderr = %q", status, stderr)
			}
			if ran(m, taskLine) != clone || ran(m, bootstrapLine) == clone {
				t.Errorf("calls = %q", m.Calls)
			}
		})
	}
}

func TestSetupSecretsStopsWhereTheTaskFails(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	m.Results[taskLine] = effecttest.Result{Err: errors.New("secrets: no host key")}
	status, stderr := runSetup(p, rel, "--secrets")
	if status != 1 || !strings.Contains(stderr, "no host key") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

func TestSetupPlanSecretsPlansThroughTheTask(t *testing.T) {
	for name, tc := range map[string]struct {
		args   []string
		clone  bool
		result effecttest.Result
		status int
	}{
		"the flag":        {[]string{"--plan", "--secrets"}, false, effecttest.Result{}, 0},
		"a clone":         {[]string{"--plan"}, true, effecttest.Result{}, 0},
		"a change":        {[]string{"--plan", "--secrets"}, false, effecttest.Result{Err: miseExit(2)}, 2},
		"a failed task":   {[]string{"--plan", "--secrets"}, false, effecttest.Result{Err: miseExit(1)}, 1},
		"a clone, a plan": {[]string{"--plan"}, true, effecttest.Result{Err: miseExit(2)}, 2},
	} {
		t.Run(name, func(t *testing.T) {
			rel := setupRelease(miseBody)
			p, m := setupProcess(rel)
			standing(m, rel)
			m.Results[taskPlanLine] = tc.result
			if tc.clone {
				m.Dirs[cloneDir] = 0o700
			}
			if status, _ := runSetup(p, rel, tc.args...); status != tc.status {
				t.Errorf("status = %d, want %d", status, tc.status)
			}
			if !ran(m, taskPlanLine) || ran(m, planLine) || ran(m, taskLine) {
				t.Errorf("calls = %q", m.Calls)
			}
		})
	}
}

func TestSetupPlanLeavesTheTaskOutWhereAStepIsDue(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	standing(m, rel)
	m.Results["git -C /opt/ws rev-parse --verify -q HEAD^{commit}"] = effecttest.Result{Out: []byte("old\n")}
	if status, stderr := runSetup(p, rel, "--plan", "--secrets"); status != 2 || !strings.Contains(stderr, "would check out v1.2.3") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if ran(m, taskPlanLine) || ran(m, taskLine) {
		t.Errorf("the task ran: %q", m.Calls)
	}
}
