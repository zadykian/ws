package claude

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/values"
)

// Marketplaces adds claude's plugin marketplaces for the user. claude plugin marketplace add takes
// a source and no name: a marketplace bears the name its own marketplace.json gives. So it adds
// each name claude does not list, and fails where the marketplace added names itself otherwise, as
// each run would add it again. add clones a git source and declares the marketplace in claude's
// user settings; it needs no claude login.
type Marketplaces struct {
	M    effect.Machine
	User host.User
	List []values.Marketplace
}

func (p Marketplaces) claude() string { return p.User.Home + "/.local/bin/claude" }

// cli runs claude plugin marketplace as the user. CLAUDE_CONFIG_DIR is unset, as the caller's
// would have claude keep its settings and plugins there, not in ~/.claude. Set to ~/.claude, it
// would move claude's .claude.json there too.
func (p Marketplaces) cli(ctx context.Context, args ...string) ([]byte, error) {
	argv := append([]string{"-u", "CLAUDE_CONFIG_DIR", p.claude(), "plugin", "marketplace"}, args...)
	return p.User.Output(ctx, p.M, "env", argv...)
}

// listed is a marketplace as claude lists it: a GitHub repository's repo, a git URL's url or a path.
type listed struct {
	Name string `json:"name"`
	Repo string `json:"repo"`
	URL  string `json:"url"`
	Path string `json:"path"`
	Ref  string `json:"ref"`
}

// source rebuilds the source that add takes: the repo, URL or path, with its ref apart.
func (l listed) source() string {
	s := cmp.Or(l.Repo, l.URL, l.Path)
	if l.Ref != "" {
		s += "#" + l.Ref
	}
	return s
}

// bare drops what claude drops or adds when it lists a source: a trailing slash, and the .git a
// GitHub URL gains, so that two sources compare without either. The ref is kept.
func bare(source string) string {
	base, ref, hasRef := strings.Cut(source, "#")
	base = strings.TrimSuffix(strings.TrimRight(base, "/"), ".git")
	if hasRef {
		return base + "#" + ref
	}
	return base
}

func (p Marketplaces) list(ctx context.Context) ([]listed, error) {
	out, err := p.cli(ctx, "list", "--json")
	if err != nil {
		return nil, err
	}
	var all []listed
	if len(strings.TrimSpace(string(out))) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(out, &all); err != nil {
		return nil, fmt.Errorf("claude plugin marketplace list: %w", err)
	}
	return all, nil
}

// checkSource fails where claude lists the marketplace from another source than the list gives.
func checkSource(all []listed, mk values.Marketplace) error {
	for _, l := range all {
		if l.Name == mk.Name && bare(l.source()) != bare(mk.Source) {
			return fmt.Errorf("claude lists the marketplace %s from %s, not %s. Give that source, or remove the marketplace first, with claude plugin marketplace remove %s",
				mk.Name, l.source(), mk.Source, mk.Name)
		}
	}
	return nil
}

