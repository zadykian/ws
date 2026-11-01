package wizard

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// hookMacStage names the stage in its messages.
const hookMacStage = "Hook on the Mac"

// HookMac is the stage of the repository that mac.conf's WS_HOOK names as OWNER/REPO: its clone in
// ~/repository/REPO by gh's login, core.hooksPath at its .githooks, its Brewfile's formulae.
type HookMac struct {
	// Host reads mac.conf, and Gh finds gh, whose login clones.
	Host *HostKey
	Gh   *Gh
	Home string
	M    effect.Machine
	Out  effect.Out
}

// hookRun is the change that runs a command.
type hookRun struct {
	text string
	argv []string
}

func (c hookRun) String() string { return hookMacStage + ": would run " + c.text }

// hookClone is the change that clones the repository.
type hookClone struct{ hookRun }

func (c hookClone) String() string {
	return hookMacStage + ": would run " + c.text + ", then set its core.hooksPath and run brew bundle, " +
		"where it has .githooks or a Brewfile"
}

// originOf returns the pattern of WS_HOOK's origin URL on GitHub, by https or ssh, letter case aside.
func originOf(hook string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)[/@]github\.com[/:]` + regexp.QuoteMeta(strings.TrimSuffix(hookRepo(hook), ".git")) + `$`)
}

// plan returns the changes that the clone lacks. A path with no clone, or another's, stops it.
func (h *HookMac) plan(ctx context.Context, hook string) ([]step.Change, error) {
	repo := hookRepo(hook)
	dir := h.Home + "/repository/" + repo[strings.Index(repo, "/")+1:]
	info, err := h.M.Stat(dir)
	if errors.Is(err, fs.ErrNotExist) && info.Link == "" {
		path, _, err := h.Gh.ghPath(ctx)
		if err != nil {
			return nil, err
		}
		argv := []string{path, "repo", "clone", repo, dir, "--", "-q"}
		return []step.Change{hookClone{hookRun{"gh repo clone " + repo + " " + dir, argv}}}, nil
	}
	if _, err := h.M.Stat(dir + "/.git"); err != nil {
		return nil, fmt.Errorf("%s: %s holds no clone: move it aside", hookMacStage, dir)
	}
	url, _ := h.M.Run(ctx, "git", "-C", dir, "remote", "get-url", "origin")
	u := strings.TrimSuffix(strings.TrimSpace(string(url)), ".git")
	if !originOf(hook).MatchString(u) {
		return nil, fmt.Errorf("%s: %s is not %s's clone, its origin '%s': move it aside", hookMacStage, dir, hookRepo(hook), u)
	}
	var cs []step.Change
	if _, err := h.M.Stat(dir + "/.githooks"); err == nil {
		got, _ := h.M.Run(ctx, "git", "-C", dir, "config", "core.hooksPath")
		if strings.TrimSpace(string(got)) != dir+"/.githooks" {
			cs = append(cs, hookRun{"git -C " + dir + " config core.hooksPath " + dir + "/.githooks",
				[]string{"git", "-C", dir, "config", "core.hooksPath", dir + "/.githooks"}})
		}
	}
	brewfile := dir + "/Brewfile"
	if _, err := h.M.Stat(brewfile); err == nil {
		// No auto-update, which brew bundle would run first, writing under --check too.
		if _, err := h.M.Run(ctx, "env", "HOMEBREW_NO_AUTO_UPDATE=1", "brew", "bundle", "check",
			"--no-upgrade", "--file="+brewfile); err != nil {
			cs = append(cs, hookRun{"brew bundle --no-upgrade --file=" + brewfile,
				[]string{"brew", "bundle", "--no-upgrade", "--file=" + brewfile}})
		}
	}
	return cs, nil
}

// Plan is empty where mac.conf names no repository.
func (h *HookMac) Plan(ctx context.Context) (step.Plan, error) {
	c, err := h.Host.conf()
	if err != nil || c.Hook == "" {
		return step.Plan{}, err
	}
	cs, err := h.plan(ctx, c.Hook)
	return step.Plan{Changes: cs}, err
}

// hookRepo returns OWNER/REPO of WS_HOOK's value, which is OWNER/REPO or OWNER/REPO@TAG.
func hookRepo(v string) string {
	repo, _, _ := strings.Cut(v, "@")
	return repo
}

// hookTag returns the TAG of WS_HOOK's value, or "" where it holds none.
func hookTag(v string) string {
	_, tag, _ := strings.Cut(v, "@")
	return tag
}
