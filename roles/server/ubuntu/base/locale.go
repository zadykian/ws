package base

import (
	"context"
	"fmt"
	"regexp"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
)

// localeParts keeps the locale generated and the system's. /etc/locale.gen names it, and
// locale-gen makes it. /etc/locale.conf, where Ubuntu 26.04 keeps the system's locale, sets LANG
// to it. /etc/default/locale links to that file.
func localeParts(m effect.Machine, name string) []resource.Resource {
	return []resource.Resource{
		generated{M: m, Name: name},
		file.Line{M: m, Path: "/etc/locale.conf", Text: "LANG=" + name, Match: "^LANG=", Mode: 0o644},
	}
}

// generated is a locale that locale-gen has made, as community.general.locale_gen kept it.
type generated struct {
	M    effect.Machine
	Name string
}

type generate struct{ g generated }

func (generate) String() string { return "generate the locale with locale-gen" }

func (c generate) Apply(ctx context.Context) error {
	_, err := c.g.M.RunRoot(ctx, "locale-gen")
	return err
}

// Plan checks the name against the locales glibc can generate, as the module did, and plans
// nothing where `locale -a` lists the locale. Else it enables the line in /etc/locale.gen and
// plans locale-gen.
func (g generated) Plan(ctx context.Context) ([]resource.Change, error) {
	supported, err := g.M.ReadFile(supportedFile)
	if err != nil {
		return nil, fmt.Errorf("base_locale %q: reading %s: %w", g.Name, supportedFile, err)
	}
	line, ok := entry(string(supported), g.Name)
	if !ok {
		return nil, fmt.Errorf("base_locale is %q: %s lacks it; it takes a name such as en_US.UTF-8", g.Name, supportedFile)
	}
	have, err := g.M.Run(ctx, "locale", "-a")
	if err != nil {
		return nil, err
	}
	if listed(string(have), g.Name) {
		return nil, nil
	}
	cs, err := file.Line{M: g.M, Path: "/etc/locale.gen", Text: line, Match: `^#?\s*` + regexp.QuoteMeta(g.Name) + `\s`, Mode: 0o644}.Plan(ctx)
	if err != nil {
		return nil, err
	}
	return append(cs, generate{g}), nil
}
