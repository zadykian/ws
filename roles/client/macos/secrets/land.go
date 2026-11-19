package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
)

// Change is a change of a clone's files, which Land lands on main.
type Change struct {
	// Topic names the branch, ws/TOPIC-YYYYMMDD-HHMMSS, as init or add-HOSTNAME.
	Topic string
	// Title and Body are the commit's and the pull request's. They hold no value.
	Title, Body string
	// Files are the paths in the clone that the change stages, and no other.
	Files []string
}

const (
	// ChecksWait bounds Land's wait for the checks, which take a minute or two on GitHub.
	ChecksWait = 15 * time.Minute
	// checksEvery is how often Land reads the checks, as gh's --watch does.
	checksEvery = 10 * time.Second
)

// checks are the secrets repository's jobs that its ruleset requires before a merge.
var checks = []string{"values", "gitleaks"}

// Land commits the change's files on a branch of their own, opens a pull request of it, waits for
// its checks, and merges it by a merge commit. The ruleset on main refuses a push to it, and
// requires the checks. It returns the pull request's URL, or "" where the files hold no change: no
// branch and no pull request then. A failed check stops it with the URL, and leaves the pull
// request open.
func (c *Clone) Land(ctx context.Context, out io.Writer, ch Change) (string, error) {
	if ch.Topic == "" || ch.Title == "" || len(ch.Files) == 0 {
		return "", errors.New("a change of the secrets repository needs a topic, a title and its files")
	}
	git := func(args ...string) ([]byte, error) {
		return c.M.Run(ctx, "git", append([]string{"-C", c.Dir}, args...)...)
	}
	status, err := git(append([]string{"status", "--porcelain=v1", "--"}, ch.Files...)...)
	if err != nil {
		return "", fmt.Errorf("cannot read the clone's status: %w", err)
	}
	if strings.TrimSpace(string(status)) == "" {
		return "", nil
	}
	branch := "ws/" + ch.Topic + "-" + c.now().UTC().Format("20060102-150405")
	commit := []string{"commit", "-q", "-S", "-m", ch.Title}
	if ch.Body != "" {
		commit = append(commit, "-m", ch.Body)
	}
	for _, args := range [][]string{{"switch", "-q", "-c", branch}, append([]string{"add", "--"}, ch.Files...), commit} {
		if _, err := git(args...); err != nil {
			return "", fmt.Errorf("cannot commit the change on %s: %w", branch, err)
		}
	}
	head, err := git("rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("cannot read the commit of %s: %w", branch, err)
	}
	if _, err := c.M.Run(ctx, "env", "GIT_TERMINAL_PROMPT=0", "git", "-C", c.Dir, "push", "-q", "-u", "origin",
		branch); err != nil {
		return "", fmt.Errorf("cannot push %s to %s: %w", branch, c.Repo.Slug(), err)
	}
	url, err := c.openPull(ctx, ch, branch)
	if err != nil {
		return "", err
	}
	_, _ = fmt.Fprintf(out, "opened %s\n", url)
	if err := c.waitChecks(ctx, out, url); err != nil {
		return url, err
	}
	// The head's SHA keeps a merge from taking a commit pushed to the branch after the checks.
	if _, err := c.M.Run(ctx, GH, "pr", "merge", url, "--merge", "--delete-branch", "--match-head-commit",
		strings.TrimSpace(string(head))); err != nil {
		return url, fmt.Errorf("cannot merge %s: %w", url, err)
	}
	_, _ = fmt.Fprintf(out, "merged %s\n", url)
	return url, nil
}

// openPull opens the pull request of the pushed branch, and returns its URL, the last line of
// gh's output that is one.
func (c *Clone) openPull(ctx context.Context, ch Change, branch string) (string, error) {
	out, err := c.M.Run(ctx, GH, "pr", "create", "--repo", c.Repo.Slug(), "--base", "main", "--head", branch,
		"--title", ch.Title, "--body", ch.Body)
	if err != nil {
		return "", fmt.Errorf("cannot open a pull request of %s: %w", branch, err)
	}
	url := ""
	for line := range strings.Lines(string(out)) {
		if line = strings.TrimSpace(line); strings.HasPrefix(line, "https://") {
			url = line
		}
	}
	if url == "" {
		return "", fmt.Errorf("gh named no pull request of %s", branch)
	}
	return url, nil
}

// waitChecks reads the pull request's checks every checksEvery until each one passes, one fails or
// ChecksWait ends.
func (c *Clone) waitChecks(ctx context.Context, out io.Writer, url string) error {
	_, _ = fmt.Fprintf(out, "waiting for the checks %s of %s\n", strings.Join(checks, " and "), url)
	start := c.now()
	for {
		passed, err := c.checksPassed(ctx, url)
		if err != nil || passed {
			return err
		}
		if c.now().Sub(start) >= ChecksWait {
			return fmt.Errorf("the checks of %s have not ended in %v: merge it once they pass", url, ChecksWait)
		}
		if err := c.sleep(ctx, checksEvery); err != nil {
			return fmt.Errorf("stopped waiting for the checks of %s: %w", url, err)
		}
	}
}

// checksPassed reads the checks once. Right after the pull request opens, gh reports none, and
// fails so: that is a check still to come. Each check's every run must pass, and a re-run's failure
// stops it as the first run's does.
func (c *Clone) checksPassed(ctx context.Context, url string) (bool, error) {
	out, err := c.M.Run(ctx, GH, "pr", "checks", url, "--json", "name,bucket")
	if err != nil {
		if strings.Contains(err.Error(), "no checks reported") {
			return false, nil
		}
		return false, fmt.Errorf("cannot read the checks of %s: %w", url, err)
	}
	var runs []struct {
		Name   string `json:"name"`
		Bucket string `json:"bucket"`
	}
	if err := json.Unmarshal(out, &runs); err != nil {
		return false, fmt.Errorf("cannot read the checks of %s: %w", url, err)
	}
	passed := map[string]bool{}
	for _, r := range runs {
		if !slices.Contains(checks, r.Name) {
			continue
		}
		switch r.Bucket {
		case "fail", "cancel":
			return false, fmt.Errorf("the check %s failed: %s", r.Name, url)
		case "pass":
			if _, seen := passed[r.Name]; !seen {
				passed[r.Name] = true
			}
		default:
			passed[r.Name] = false
		}
	}
	for _, name := range checks {
		if !passed[name] {
			return false, nil
		}
	}
	return true, nil
}

func (c *Clone) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Clone) sleep(ctx context.Context, d time.Duration) error {
	if c.Sleep != nil {
		return c.Sleep(ctx, d)
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
