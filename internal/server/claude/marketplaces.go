package claude

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
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

type addMarketplace struct {
	p  Marketplaces
	mk values.Marketplace
}

func (c addMarketplace) String() string {
	return "add the plugin marketplace " + c.mk.Name + " from " + c.mk.Source
}

// Apply adds the marketplace. --json ends the output with a line of JSON, which names the
// marketplace as it names itself. claude keeps it all the same, under its own name. It lists the
// marketplaces after, as claude lists some sources in another form than add took them, such as a
// relative path made absolute. The next run's check would fail on such a one, so this run fails now.
func (c addMarketplace) Apply(ctx context.Context) error {
	out, err := c.p.cli(ctx, "add", "--json", c.mk.Source)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var added struct {
		Marketplace string `json:"marketplace"`
	}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &added); err != nil {
		return fmt.Errorf("claude plugin marketplace add: %w", err)
	}
	if added.Marketplace != c.mk.Name {
		return fmt.Errorf("the marketplace from %s names itself %s, not %s. claude has added it as %s: give that name in claude_marketplaces",
			c.mk.Source, added.Marketplace, c.mk.Name, added.Marketplace)
	}
	all, err := c.p.list(ctx)
	if err != nil {
		return err
	}
	return checkSource(all, c.mk)
}

// Plan lists the marketplaces claude knows, checks each known one against its source, and plans an
// add for each other. Without claude, as in a dry run on a new machine, it lists none, so the plan
// names every marketplace.
func (p Marketplaces) Plan(ctx context.Context) ([]host.Change, error) {
	var all []listed
	if _, err := p.M.Stat(p.claude()); err == nil {
		if all, err = p.list(ctx); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	var changes []host.Change
	for _, mk := range p.List {
		known := false
		for _, l := range all {
			known = known || l.Name == mk.Name
		}
		if !known {
			changes = append(changes, addMarketplace{p, mk})
		} else if err := checkSource(all, mk); err != nil {
			return nil, err
		}
	}
	return changes, nil
}
