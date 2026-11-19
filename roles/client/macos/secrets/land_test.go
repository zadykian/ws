package secrets_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/secrets"
)

// The lines that land a change of fnox.toml in the clone of o/n, at 18:00:00 UTC.
const (
	branch     = "ws/init-20261119-180000"
	statusLine = "git -C " + cloneDir + " status --porcelain=v1 -- fnox.toml"
	switchLine = "git -C " + cloneDir + " switch -q -c " + branch
	addLine    = "git -C " + cloneDir + " add -- fnox.toml"
	commitLine = "git -C " + cloneDir + " commit -q -S -m chore: set up -m The recipients."
	headLine   = "git -C " + cloneDir + " rev-parse HEAD"
	pushLine   = "env GIT_TERMINAL_PROMPT=0 git -C " + cloneDir + " push -q -u origin " + branch
	createLine = secrets.GH + " pr create --repo o/n --base main --head " + branch +
		" --title chore: set up --body The recipients."
	pullURL    = "https://github.com/o/n/pull/7"
	checksLine = secrets.GH + " pr checks " + pullURL + " --json name,bucket"
	mergeLine  = secrets.GH + " pr merge " + pullURL + " --merge --delete-branch --match-head-commit abc123"
	bothGreen  = `[{"name":"values","bucket":"pass"},{"name":"lint","bucket":"fail"},{"name":"gitleaks","bucket":"pass"}]`
	oneRunning = `[{"name":"values","bucket":"pass"},{"name":"gitleaks","bucket":"pending"}]`
)

var change = secrets.Change{Topic: "init", Title: "chore: set up", Body: "The recipients.", Files: []string{"fnox.toml"}}

// clock is Land's clock: each sleep moves it on, and runs the test's turn where it has one.
type clock struct {
	now   time.Time
	slept int
	turn  func(slept int)
}

func (k *clock) Now() time.Time { return k.now }

func (k *clock) Sleep(_ context.Context, d time.Duration) error {
	k.now = k.now.Add(d)
	k.slept++
	if k.turn != nil {
		k.turn(k.slept)
	}
	return nil
}

// landable opens o/n on a Mac where each line of a change's landing passes and both checks pass.
// It returns the calls Open made, which Land's come after.
func landable(t *testing.T) (*effecttest.Machine, *secrets.Clone, *clock, int) {
	m := cloneable()
	for line, out := range map[string]string{
		statusLine: " M fnox.toml\n", switchLine: "", addLine: "", commitLine: "", headLine: "abc123\n",
		pushLine: "", createLine: "Creating pull request\n" + pullURL + "\n", checksLine: bothGreen, mergeLine: "",
	} {
		m.Results[line] = effecttest.Result{Out: []byte(out)}
	}
	c := open(t, m)
	k := &clock{now: time.Date(2026, 11, 19, 18, 0, 0, 0, time.UTC)}
	c.Now, c.Sleep = k.Now, k.Sleep
	return m, c, k, len(m.Calls)
}

func TestLandOpensNothingWithoutAChange(t *testing.T) {
	m, c, _, _ := landable(t)
	m.Results[statusLine] = effecttest.Result{}
	url, err := c.Land(context.Background(), &strings.Builder{}, change)
	if url != "" || err != nil {
		t.Fatalf("Land of no change gives %q, %v, want nothing", url, err)
	}
	for _, line := range []string{switchLine, pushLine, createLine} {
		if slices.Contains(m.Calls, "run "+line) {
			t.Errorf("Land of no change ran %s", line)
		}
	}
}

func TestLandMergesOnceBothChecksPass(t *testing.T) {
	m, c, k, opened := landable(t)
	m.Results[checksLine] = effecttest.Result{Err: errors.New("[gh pr checks]: exit status 1: " +
		"no checks reported on the '" + branch + "' branch")}
	k.turn = func(slept int) {
		m.Results[checksLine] = effecttest.Result{Out: []byte(map[int]string{1: oneRunning, 2: bothGreen}[slept])}
	}
	var out strings.Builder
	url, err := c.Land(context.Background(), &out, change)
	if url != pullURL || err != nil {
		t.Fatalf("Land gives %q, %v, want %s", url, err, pullURL)
	}
	var want []string
	for _, line := range []string{statusLine, switchLine, addLine, commitLine, headLine, pushLine, createLine,
		checksLine, checksLine, checksLine, mergeLine} {
		want = append(want, "run "+line)
	}
	if got := m.Calls[opened:]; !slices.Equal(got, want) {
		t.Errorf("Land ran\n%q\nwant\n%q", got, want)
	}
	if k.slept != 2 {
		t.Errorf("Land slept %d times, want 2", k.slept)
	}
	if want := "opened " + pullURL + "\nwaiting for the checks values and gitleaks of " + pullURL +
		"\nmerged " + pullURL + "\n"; out.String() != want {
		t.Errorf("Land printed %q, want %q", out.String(), want)
	}
}

func TestLandStopsOnAFailedCheck(t *testing.T) {
	for _, bucket := range []string{"fail", "cancel"} {
		m, c, _, _ := landable(t)
		m.Results[checksLine] = effecttest.Result{Out: []byte(`[{"name":"values","bucket":"pass"},` +
			`{"name":"gitleaks","bucket":"` + bucket + `"}]`)}
		url, err := c.Land(context.Background(), &strings.Builder{}, change)
		if url != pullURL || err == nil || !strings.Contains(err.Error(), "gitleaks") ||
			!strings.Contains(err.Error(), pullURL) {
			t.Errorf("a check of %s gives %q, %v, want an error naming gitleaks and the URL", bucket, url, err)
		}
		if slices.Contains(m.Calls, "run "+mergeLine) {
			t.Errorf("a check of %s merged", bucket)
		}
	}
}

func TestLandStopsWhereTheChecksOutlastTheWait(t *testing.T) {
	m, c, k, _ := landable(t)
	m.Results[checksLine] = effecttest.Result{Out: []byte(oneRunning)}
	_, err := c.Land(context.Background(), &strings.Builder{}, change)
	if err == nil || !strings.Contains(err.Error(), pullURL) || !strings.Contains(err.Error(), "15m0s") {
		t.Errorf("checks past the wait give %v, want an error naming the URL and 15m0s", err)
	}
	if k.slept != 90 || slices.Contains(m.Calls, "run "+mergeLine) {
		t.Errorf("Land slept %d times and ran %q, want 90 sleeps and no merge", k.slept, m.Calls)
	}
}

func TestLandNamesThePullRequestWhereTheMergeFails(t *testing.T) {
	m, c, _, _ := landable(t)
	m.Results[mergeLine] = effecttest.Result{Err: errors.New("[gh pr merge]: exit status 1: not mergeable")}
	url, err := c.Land(context.Background(), &strings.Builder{}, change)
	if url != pullURL || err == nil || !strings.Contains(err.Error(), pullURL) {
		t.Errorf("a failed merge gives %q, %v, want an error naming %s", url, err, pullURL)
	}
}

func TestLandStagesTheNamedFilesAlone(t *testing.T) {
	m, c, k, _ := landable(t)
	k.now = time.Date(2026, 11, 19, 21, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60))
	if _, err := c.Land(context.Background(), &strings.Builder{}, change); err != nil {
		t.Fatal(err)
	}
	var adds []string
	for _, call := range m.Calls {
		if strings.Contains(call, " add ") {
			adds = append(adds, call)
		}
	}
	if !slices.Equal(adds, []string{"run " + addLine}) || !slices.Contains(m.Calls, "run "+switchLine) {
		t.Errorf("Land staged %q on %q, want fnox.toml alone on %s", adds, m.Calls, branch)
	}
}
