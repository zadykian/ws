package cli

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/internal/mactest"
	"github.com/zadykian/ws/roles/client/values"
	"github.com/zadykian/ws/roles/internal/setup"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

const (
	installLine = "/usr/local/bin/mise -C /opt/ws/roles/client bootstrap packages apply --yes macos-app:secretive"
	spctlLine   = "sh -c " + spctlScript + " sh " + secretiveApp
	openLine    = "open " + secretiveApp
	accepted    = secretiveApp + ": accepted\nsource=Notarized Developer ID\n" +
		"origin=Developer ID Application: Max Goedjen (Z72PRUAWF6)\n"
	noKeys = "The agent has no identities.\n"
	// mldsaKey is a key of a type that ws's OpenSSH lacks.
	mldsaKey = "ssh-mldsa-65 CCCC me\n"
)

// unkeyed returns the process of a Mac whose values.yml names no login key. Secretive's agent lists
// no key at its first look, and the list then is what the agent gives.
func unkeyed(rel setup.MiseRelease, then string) (enginecli.Env, *effecttest.Machine) {
	p, m := setupProcess(rel)
	writeConfs(m, values.Text("WS_GIT_SIGNING_PUB_KEY", "~/g.pub"), values.Text("WS_SLOT", "0"))
	mactest.AgentMachine(m)
	m.Results[mactest.AgentList] = effecttest.Result{Out: []byte(noKeys), Err: errors.New("exit status 1")}
	m.After = map[string]map[string]effecttest.Result{
		mactest.AgentList: {mactest.AgentList: {Out: []byte(then)}}}
	m.Results[installLine] = effecttest.Result{}
	m.Results[spctlLine] = effecttest.Result{Out: []byte(accepted)}
	m.Results[openLine] = effecttest.Result{}
	return p, m
}

// exactly is the place of the call that is the line, as indexOf finds a line by its prefix, and the
// bootstrap's line starts the install's.
func exactly(m *effecttest.Machine, call string) int { return slices.Index(m.Calls, call) }

func valuesText(m *effecttest.Machine) string { return string(m.Files["/etc/ws/values.yml"]) }

func TestSetupTakesTheKeyThatComesDuringTheWaitAndNamesIt(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := unkeyed(rel, mactest.AgentKey)
	status, stderr := runSetup(p, rel)
	if status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	held := mactest.CopyOf(mactest.AgentKey)
	// The signing key that values.yml names stays, and the login key and its agent are new.
	want := "WS_GIT_SIGNING_PUB_KEY: ~/g.pub\nWS_IDENTITY_PUB_KEY: " + held + "\nWS_IDENTITY_AGENT: " + secretiveAgent + "\n"
	if got := valuesText(m); got != want {
		t.Errorf("values.yml = %q, want %q", got, want)
	}
	last := -1
	for _, call := range []string{"run " + installLine, "run " + spctlLine, "run " + mactest.AgentList, "run " + openLine,
		"atomic /etc/ws/values.yml 644", "run " + miseLine} {
		at := exactly(m, call)
		if at <= last {
			t.Errorf("%q is at %d, after %d: %q", call, at, last, m.Calls)
		}
		last = at
	}
	if !strings.Contains(stderr, "make one at Notify") {
		t.Errorf("the wait said nothing of Notify: %q", stderr)
	}
}

func TestSetupNamesTheSigningKeyToo(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := unkeyed(rel, mactest.AgentKey)
	writeConfs(m, "", values.Text("WS_SLOT", "0"))
	if status, stderr := runSetup(p, rel); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	held := mactest.CopyOf(mactest.AgentKey)
	if got := valuesText(m); !strings.Contains(got, "WS_GIT_SIGNING_PUB_KEY: "+held+"\n") {
		t.Errorf("values.yml = %q, want the key for git's signing too", got)
	}
}

func TestSetupTakesNoKeyThatWasThereBeforeTheWait(t *testing.T) {
	rel := setupRelease(miseBody)
	_, m := unkeyed(rel, mactest.AgentKey)
	// The agent lists the key at its first look already.
	m.Results[mactest.AgentList] = effecttest.Result{Out: []byte(mactest.AgentKey)}
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	var out effecttest.Out
	s := secretiveKey{M: m, Out: &out, Home: mactest.AgentHome, Values: "/etc/ws/values.yml", Poll: time.Millisecond}
	changes, err := s.Plan(ctx)
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	err = changes[0].Apply(ctx)
	if err == nil || !strings.Contains(err.Error(), "stopped before Secretive held a key") {
		t.Errorf("err = %v, want the stop of the wait", err)
	}
	if said := strings.Join(out.Lines, "\n"); !strings.Contains(said, "--key "+"NAME") || !strings.Contains(said, "Ctrl-C and") ||
		!strings.Contains(said, strings.TrimSuffix(sshName(mactest.AgentKey), ".pub")) {
		t.Errorf("the wait did not offer the key by --key: %q", out.Lines)
	}
	if indexOf(m, "atomic /etc/ws/values.yml") >= 0 {
		t.Errorf("values.yml was written: %q", m.Calls)
	}
}

func TestSetupTakesTheKeyThatKeyNames(t *testing.T) {
	rel := setupRelease(miseBody)
	for name, pick := range map[string]string{
		"its name":        strings.TrimSuffix(sshName(mactest.AgentGit), ".pub"),
		"its name by pub": sshName(mactest.AgentGit),
		"its fingerprint": sshcfg.AgentKey{Line: strings.TrimSpace(mactest.AgentGit)}.Fingerprint(),
	} {
		t.Run(name, func(t *testing.T) {
			p, m := unkeyed(rel, mactest.AgentKey+mactest.AgentGit)
			m.Results[mactest.AgentList] = effecttest.Result{Out: []byte(mactest.AgentKey + mactest.AgentGit)}
			if status, stderr := runSetup(p, rel, "--key", pick); status != 0 {
				t.Fatalf("status = %d, stderr = %q", status, stderr)
			}
			if got := valuesText(m); !strings.Contains(got, "WS_IDENTITY_PUB_KEY: "+mactest.CopyOf(mactest.AgentGit)+"\n") {
				t.Errorf("values.yml = %q, want the key that --key named", got)
			}
		})
	}
}

func TestSetupSaysWhatSecretiveHoldsWhereNoKeyIsNamedByKey(t *testing.T) {
	rel := setupRelease(miseBody)
	_, m := unkeyed(rel, mactest.AgentKey)
	m.Results[mactest.AgentList] = effecttest.Result{Out: []byte(mactest.AgentKey)}
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	var out effecttest.Out
	s := secretiveKey{M: m, Out: &out, Home: mactest.AgentHome, Values: "/etc/ws/values.yml", Poll: time.Millisecond,
		Pick: "nosuch"}
	changes, err := s.Plan(ctx)
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if err = changes[0].Apply(ctx); err == nil || !strings.Contains(err.Error(), "stopped before Secretive held a key") {
		t.Errorf("err = %v, want the stop of the wait", err)
	}
	said := strings.Join(out.Lines, "\n")
	if !strings.Contains(said, "no key that nosuch names") ||
		!strings.Contains(said, strings.TrimSuffix(sshName(mactest.AgentKey), ".pub")) {
		t.Errorf("the wait did not say what Secretive holds: %q", out.Lines)
	}
	if n := strings.Count(said, "no key that nosuch names"); n != 1 {
		t.Errorf("the wait said it %d times, want once: %q", n, out.Lines)
	}
}

func TestSetupRefusesSeveralKeysThatComeAtOnce(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := unkeyed(rel, mactest.AgentKey+mactest.AgentGit)
	status, stderr := runSetup(p, rel)
	if status != 1 || !strings.Contains(stderr, "2 keys came at once") || !strings.Contains(stderr, "--key NAME") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if indexOf(m, "atomic /etc/ws/values.yml") >= 0 || exactly(m, "run "+miseLine) >= 0 {
		t.Errorf("setup went on: %q", m.Calls)
	}
}

func TestSetupSkipsAKeyOfAnotherTypeAndRefusesItByKey(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := unkeyed(rel, mldsaKey+mactest.AgentKey)
	if status, stderr := runSetup(p, rel); status != 0 || !strings.Contains(valuesText(m), mactest.CopyOf(mactest.AgentKey)) {
		t.Errorf("status = %d, stderr = %q, values.yml = %q", status, stderr, valuesText(m))
	}
	p, m = unkeyed(rel, mldsaKey)
	m.Results[mactest.AgentList] = effecttest.Result{Out: []byte(mldsaKey)}
	status, stderr := runSetup(p, rel, "--key", strings.TrimSuffix(sshName(mldsaKey), ".pub"))
	if status != 1 || !strings.Contains(stderr, "not a P-256 one") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

func TestSetupStopsWhereMacOSDoesNotAcceptSecretive(t *testing.T) {
	rel := setupRelease(miseBody)
	for name, res := range map[string]effecttest.Result{
		"rejected":      {Out: []byte(secretiveApp + ": rejected\n"), Err: errors.New("exit status 3")},
		"not notarized": {Out: []byte(secretiveApp + ": accepted\nsource=Developer ID\n")},
		"another team": {Out: []byte(secretiveApp + ": accepted\nsource=Notarized Developer ID\n" +
			"origin=Developer ID Application: Someone Else (AB12CD34EF)\n")},
		"no team": {Out: []byte(secretiveApp + ": accepted\nsource=Notarized Developer ID\n")},
		"the team in the name alone": {Out: []byte(secretiveApp + ": accepted\nsource=Notarized Developer ID\n" +
			"origin=Developer ID Application: Max Goedjen (Z72PRUAWF6) (AB12CD34EF)\n")},
	} {
		t.Run(name, func(t *testing.T) {
			p, m := unkeyed(rel, mactest.AgentKey)
			m.Results[spctlLine] = res
			status, stderr := runSetup(p, rel)
			if status != 1 || !strings.Contains(stderr, "does not accept "+secretiveApp) {
				t.Errorf("status = %d, stderr = %q", status, stderr)
			}
			if indexOf(m, "run "+openLine) >= 0 || exactly(m, "run "+miseLine) >= 0 {
				t.Errorf("setup went on: %q", m.Calls)
			}
		})
	}
}

func TestSetupStopsWhereMiseInstallsNoSecretive(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := unkeyed(rel, mactest.AgentKey)
	m.Results[installLine] = effecttest.Result{Err: errors.New("another build stands there")}
	status, stderr := runSetup(p, rel)
	if status != 1 || !strings.Contains(stderr, "installing Secretive") || !strings.Contains(stderr, "another build stands there") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

func TestSetupWaitsForNothingWhereValuesNameTheLoginKey(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := setupProcess(rel)
	standing(m, rel)
	if status, stderr := runSetup(p, rel); status != 0 {
		t.Fatalf("status = %d, stderr = %q", status, stderr)
	}
	for _, call := range []string{"run " + installLine, "run sh -c", "run " + openLine, "run env"} {
		if indexOf(m, call) >= 0 {
			t.Errorf("Secretive's step ran %q: %q", call, m.Calls)
		}
	}
	status, stderr := runSetup(p, rel, "--key", "ab")
	if status != 1 || !strings.Contains(stderr, "names WS_IDENTITY_PUB_KEY already") {
		t.Errorf("--key with a named login key: status = %d, stderr = %q", status, stderr)
	}
}

func TestSetupPlanSaysSecretiveIsDueAndChangesNothing(t *testing.T) {
	rel := setupRelease(miseBody)
	p, m := unkeyed(rel, mactest.AgentKey)
	standing(m, rel)
	status, stderr := runSetup(p, rel, "--plan")
	if status != 2 || !strings.Contains(stderr, "would install Secretive, wait for a key made in it") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	noWrites(t, m)
	for _, call := range []string{"run " + installLine, "run " + openLine, "run " + miseLine} {
		if indexOf(m, call) >= 0 {
			t.Errorf("the plan ran %q", call)
		}
	}
}

// sshName is the name that Secretive's agent gives the key of the line, as its .pub's.
func sshName(line string) string {
	return mactest.CopyOf(line)[strings.LastIndex(mactest.CopyOf(line), "/")+1:]
}
