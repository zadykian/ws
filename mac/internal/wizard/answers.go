package wizard

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// errMissing stops Plan's walk over the keys at the first that mac.conf lacks.
var errMissing = errors.New("mac.conf lacks a key")

// Answers is the stage that gives mac.conf the values it lacks, each from the environment, under
// its key's name, or asked at the terminal. It runs before any stage that reads mac.conf.
type Answers struct {
	// Conf is mac.conf's path, and Roles the shell role's defaults, whose git identity ws's git takes.
	Conf, Roles string
	// Prog prefixes the messages of a refused answer.
	Prog string
	// Tty tells whether stdin is a terminal, which the questions need.
	Tty bool
	// Check makes Plan rehearse the stage: it asks, shows the diff of mac.conf, and writes what mac.conf
	// would be to Scratch, the file that the stages after it read.
	Check   bool
	Scratch string
	M       effect.Machine
	P       effect.Prompt
	Out     effect.Out
	// Env looks a variable up, telling one set to nothing from one that is not set.
	Env func(string) (string, bool)
}

// Name is the stage's name.
func (*Answers) Name() string { return answers }

// WaitsOn is empty: nothing is set up before mac.conf.
func (*Answers) WaitsOn() string { return "" }

type todo string

func (c todo) String() string { return string(c) }

func (a *Answers) file() confFile { return confFile{a.Conf, answers, a.M, a.Out} }

// load reads mac.conf and says each line that conf refuses, which setup.sh would stop at.
func (a *Answers) load() (string, conf.Conf, error) {
	text, err := a.file().read()
	c, errs := conf.Lines(text)
	for _, e := range errs {
		a.Out.Say("%s", e.In(a.Conf))
	}
	if err != nil || len(errs) > 0 {
		return "", c, fmt.Errorf("%s: cannot read %s; make it readable, then run the wizard again", answers, a.Conf)
	}
	return text, c, nil
}

// role gives the shell role's default of that name, as the role's file has it, or nothing.
func (a *Answers) role(name string) string {
	data, _ := a.M.ReadFile(a.Roles)
	for _, l := range strings.Split(string(data), "\n") {
		if v, ok := strings.CutPrefix(l, name+":"); ok {
			return strings.TrimLeft(v, " ")
		}
	}
	return ""
}

// sequence returns the walk over the keys, which gets the value of a key that mac.conf lacks from ask.
func (a *Answers) sequence(c conf.Conf, text string, ask func(question) (string, error)) *sequence {
	vals := map[string]string{"WS_TARGET": c.Target, "WS_LAN": c.LAN, "WS_GIT_NAME": c.GitName, "WS_REMOTE": c.Remote,
		"WS_HOOK": c.Hook}
	return &sequence{vals: vals, lacks: func(key string) bool { return !hasLine(text, key) }, ask: ask, role: a.role}
}

// Plan finds the first key that mac.conf lacks, or that mac.conf with all its keys is refused.
func (a *Answers) Plan(ctx context.Context) (step.Plan, error) {
	text, c, err := a.load()
	if err != nil {
		return step.Plan{}, err
	}
	if a.Check {
		return step.Plan{}, a.rehearse(ctx, text, c)
	}
	first := ""
	err = a.sequence(c, text, func(q question) (string, error) { first = q.key; return "", errMissing }).run()
	switch {
	case errors.Is(err, errMissing):
		return step.Plan{Changes: []step.Change{todo(fmt.Sprintf(
			"%s: would ask for the values that %s lacks, from %s on, and add them", answers, a.Conf, first))}}, nil
	case err != nil:
		return step.Plan{}, err
	}
	if _, errs := conf.Parse(text); len(errs) > 0 {
		return step.Plan{Changes: []step.Change{todo(answers + ": setup.sh would refuse " + a.Conf)}}, nil
	}
	return step.Plan{}, nil
}

// collect asks for the values that mac.conf lacks, in order, and returns their lines.
func (a *Answers) collect(ctx context.Context, text string, c conf.Conf) (string, error) {
	ask := &asker{conf: a.Conf, prog: a.Prog, env: a.Env, p: a.P, tty: a.Tty, out: a.Out}
	seq := a.sequence(c, text, func(q question) (string, error) { return ask.ask(ctx, q) })
	ask.lan = func() string { return seq.vals["WS_LAN"] }
	err := seq.run()
	return ask.lines, err
}

// refusal words conf's refusal of mac.conf with the lines.
func (a *Answers) refusal(lines string) error {
	with := ""
	if lines != "" {
		with = " with the answers"
	}
	return fmt.Errorf("%s: setup.sh would refuse %s%s, as above; fix it, then run the wizard again", answers, a.Conf, with)
}

// Apply asks for the values that mac.conf lacks, in order, and adds them to it. A mac.conf that
// setup.sh would refuse with them is left alone.
func (a *Answers) Apply(ctx context.Context, _ step.Plan) error {
	text, c, err := a.load()
	if err != nil {
		return err
	}
	lines, err := a.collect(ctx, text, c)
	if err != nil {
		return err
	}
	err = a.file().add(ctx, lines)
	if errors.Is(err, errRefused) {
		return a.refusal(lines)
	}
	return err
}

// rehearse does Apply's asking under --check, and shows the diff of mac.conf instead of writing it.
// The stages after read the file it writes to Scratch, as wizard.sh pointed WS_CONF at a file in the
// lock, so that they check against the mac.conf that the run would make.
func (a *Answers) rehearse(ctx context.Context, text string, c conf.Conf) error {
	lines, err := a.collect(ctx, text, c)
	if err != nil {
		return err
	}
	before, after, err := a.file().compose(lines)
	if errors.Is(err, errRefused) {
		return a.refusal(lines)
	}
	if err != nil {
		return err
	}
	if _, err := a.M.Stat(path.Dir(a.Conf)); err != nil {
		a.Out.Say("would make %s", path.Dir(a.Conf))
	}
	a.Out.Diff(a.Conf, []byte(before), []byte(after))
	if err := a.M.WriteFile(a.Scratch, []byte(after), 0o600); err != nil {
		return fmt.Errorf("%s: cannot write %s", answers, a.Scratch)
	}
	return nil
}

var _ step.Step = (*Answers)(nil)
