// Package answers is the wizard stage that asks for the values of mac.conf and a server's file, and
// writes them.
package answers

import (
	"context"
	"errors"
	"fmt"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"path"
	"strconv"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

// errMissing stops Plan's walk over the keys at the first that its file lacks.
var errMissing = errors.New("a file lacks a key")

// Answers is the stage that gives mac.conf and a server's file the values they lack. Each value
// comes from the environment, under its key's name, or is asked at the terminal. Each goes in the
// file of its key. The stage runs before any stage that reads them.
type Answers struct {
	// Conf is mac.conf's path, and Server the server's file's.
	Conf   string
	Server string
	// Defaults is the text of the shell role's defaults, whose git identity ws's git takes.
	Defaults []byte
	// Prog prefixes the messages of a refused answer.
	Prog string
	// Tty tells whether stdin is a terminal, which the questions need.
	Tty bool
	// Check makes Plan rehearse the stage. It asks, shows the diff of each file, and writes what
	// mac.conf and the server's file would be to Scratch and ServerScratch, which the stages after
	// it read. It copies each other server's file beside them, as the stages list the servers there.
	Check         bool
	Scratch       string
	ServerScratch string
	M             effect.Machine
	P             effect.Prompt
	Out           effect.Out
	// Env looks a variable up, telling one set to nothing from one that is not set.
	Env func(string) (string, bool)
}

// Name is the stage's name.
func (*Answers) Name() string { return stage.Answers }

// Side is the Mac: mac.conf.
func (*Answers) Side() guided.Side { return stage.Mac }

// WaitsOn is Homebrew: on a new Mac, a --check run stops there, as the install does.
func (*Answers) WaitsOn() string { return stage.Homebrew }

// part is one file of the stage: its text now, its text with the answers, and its scratch copy.
type part struct {
	file          conf.File
	scratch       string
	before, after string
	// gains tells whether the answers add lines to the file. A file with none is left alone.
	gains bool
}

func (a *Answers) file(s conf.Scope) conf.File {
	path := a.Conf
	if s == conf.ServerFile {
		path = a.Server
	}
	return conf.File{Path: path, Scope: s, Stage: stage.Answers, M: a.M, Out: a.Out}
}

// load reads both files and says each line that conf refuses, which ws client macos apply would
// stop at. It returns the text of each file, and the keys that they set.
func (a *Answers) load() (map[conf.Scope]string, conf.Conf, error) {
	mac, macConf, macErr := a.file(conf.MacFile).Load()
	server, c, serverErr := a.file(conf.ServerFile).Load()
	if err := errors.Join(macErr, serverErr); err != nil {
		return nil, c, err
	}
	c.GitName = macConf.GitName
	return map[conf.Scope]string{conf.MacFile: mac, conf.ServerFile: server}, c, nil
}

// sequence returns the walk over the keys, which gets the value of a key that its file lacks from
// ask.
func (a *Answers) sequence(c conf.Conf, text map[conf.Scope]string, place *conf.Place, ask func(conf.Question) (string, error)) *conf.Sequence {
	vals := map[string]string{"WS_TARGET": c.Target, "WS_LAN": c.LAN, "WS_GIT_NAME": c.GitName, "WS_REMOTE": c.Remote}
	return &conf.Sequence{Vals: vals, Lacks: func(key string) bool { return !conf.HasLine(text[conf.KeyScope(key)], key) }, Ask: ask, Place: place, Role: func(name string) string { return conf.Default(a.Defaults, name) }}
}

// Plan finds the first key that its file lacks, or the file that is refused with all its keys.
func (a *Answers) Plan(ctx context.Context) (step.Plan, error) {
	text, c, err := a.load()
	if err != nil {
		return step.Plan{}, err
	}
	if a.Check {
		return step.Plan{}, a.rehearse(ctx, text, c)
	}
	first := ""
	err = a.sequence(c, text, nil, func(q conf.Question) (string, error) { first = q.Key; return "", errMissing }).Run()
	switch {
	case errors.Is(err, errMissing):
		return step.Plan{Changes: []step.Change{stage.Todo(fmt.Sprintf(
			"%s: would ask for the values that %s lacks, from %s on, and add them", stage.Answers, a.file(conf.KeyScope(first)).Path, first))}}, nil
	case err != nil:
		return step.Plan{}, err
	}
	for _, s := range []conf.Scope{conf.MacFile, conf.ServerFile} {
		if _, errs := s.Parse(text[s]); len(errs) > 0 {
			return step.Plan{Changes: []step.Change{stage.Todo(stage.Answers + ": ws client macos apply would refuse " + a.file(s).Path)}}, nil
		}
	}
	return step.Plan{}, nil
}

// place returns where the server sits beside the Mac's others: the slot its file holds, else the
// smallest that no other server holds.
func (a *Answers) place(text map[conf.Scope]string, c conf.Conf) (*conf.Place, error) {
	others, err := conf.Siblings(a.M, a.Conf, a.Server)
	if err != nil {
		return nil, err
	}
	p := &conf.Place{Slot: c.TunnelSlot(), Others: others}
	if !conf.HasLine(text[conf.ServerFile], "WS_SLOT") {
		if p.Slot, err = conf.FreeSlot(others); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// collect asks for the values that the files lack, in order, and returns their lines and the
// server's WS_REMOTE.
func (a *Answers) collect(ctx context.Context, text map[conf.Scope]string, c conf.Conf, place *conf.Place) (string, string, error) {
	ask := &asker{conf: a.Conf, prog: a.Prog, env: a.Env, p: a.P, tty: a.Tty, out: a.Out}
	seq := a.sequence(c, text, place, func(q conf.Question) (string, error) { return ask.ask(ctx, q) })
	ask.lan = func() string { return seq.Vals["WS_LAN"] }
	err := seq.Run()
	return ask.lines, seq.Vals["WS_REMOTE"], err
}

// compose splits the lines by the file of their keys, and returns each file with its lines added,
// mac.conf first. A server's file that lacks WS_SLOT and gets lines gets its place first: the slot,
// then WS_PORT where it has none. That keeps servers behind one remote apart. The stage stops at a
// file refused with its lines, and says it.
func (a *Answers) compose(text map[conf.Scope]string, lines, remote string, place *conf.Place) ([]part, error) {
	var parts []part
	for _, p := range []part{{file: a.file(conf.MacFile), scratch: a.Scratch}, {file: a.file(conf.ServerFile), scratch: a.ServerScratch}} {
		var err error
		kept := p.file.Scope.Keep(lines)
		if p.file.Scope == conf.ServerFile && kept != "" && !conf.HasLine(text[conf.ServerFile], "WS_SLOT") {
			head := "WS_SLOT=" + strconv.Itoa(int(place.Slot)) + "\n"
			if !conf.HasLine(text[conf.ServerFile], "WS_PORT") {
				head += conf.Line("WS_PORT", place.Port(remote)) + "\n"
			}
			kept = head + kept
		}
		p.gains = kept != ""
		p.before, p.after, err = p.file.Compose(kept)
		if errors.Is(err, conf.ErrRefused) {
			return nil, a.refusal(p.file.Path, lines)
		}
		if err != nil {
			return nil, err
		}
		parts = append(parts, p)
	}
	return parts, nil
}

// refusal words conf's refusal of the file at path with the lines.
func (a *Answers) refusal(path, lines string) error {
	with := ""
	if lines != "" {
		with = " with the answers"
	}
	return fmt.Errorf("%s: ws client macos apply would refuse %s%s, as above; fix it, then run the wizard again", stage.Answers, path, with)
}

// Apply asks for the values that the files lack, in order, and adds each to the file of its key. A
// file that ws client macos apply would refuse with them leaves both alone.
func (a *Answers) Apply(ctx context.Context, _ step.Plan) error {
	text, c, err := a.load()
	if err != nil {
		return err
	}
	place, err := a.place(text, c)
	if err != nil {
		return err
	}
	lines, remote, err := a.collect(ctx, text, c, place)
	if err != nil {
		return err
	}
	parts, err := a.compose(text, lines, remote, place)
	if err != nil {
		return err
	}
	for _, p := range parts {
		if !p.gains {
			continue
		}
		if err := p.file.Write(ctx, p.after); err != nil {
			return err
		}
	}
	return nil
}

// rehearse does Apply's asking under --check, and shows the diff of each file instead of writing
// it. The stages after read the files it writes to the scratch paths, so that they check against
// what the run would make. The old script did the same: it pointed WS_CONF at a file in the lock.
func (a *Answers) rehearse(ctx context.Context, text map[conf.Scope]string, c conf.Conf) error {
	place, err := a.place(text, c)
	if err != nil {
		return err
	}
	lines, remote, err := a.collect(ctx, text, c, place)
	if err != nil {
		return err
	}
	parts, err := a.compose(text, lines, remote, place)
	if err != nil {
		return err
	}
	for _, p := range parts {
		if p.gains {
			if _, err := a.M.Stat(path.Dir(p.file.Path)); err != nil {
				a.Out.Say("would make %s", path.Dir(p.file.Path))
			}
			a.Out.Diff(p.file.Path, []byte(p.before), []byte(p.after))
		}
		if err := a.writeScratch(ctx, p.scratch, []byte(p.after)); err != nil {
			return err
		}
	}
	return a.copyOthers(ctx)
}

// writeScratch writes text to the scratch file at file, in a directory it makes.
func (a *Answers) writeScratch(ctx context.Context, file string, text []byte) error {
	if _, err := a.M.Run(ctx, "mkdir", "-p", path.Dir(file)); err != nil {
		return fmt.Errorf("%s: cannot make %s", stage.Answers, path.Dir(file))
	}
	if err := a.M.WriteFile(file, text, 0o600); err != nil {
		return fmt.Errorf("%s: cannot write %s", stage.Answers, file)
	}
	return nil
}

// copyOthers copies the file of each other server beside mac.conf, as it stands, to the scratch's
// servers/. The run reads them beside the server's file. So a refused one or a clash refuses the
// rehearsal too, and the steps plan for the servers that the run would list.
func (a *Answers) copyOthers(ctx context.Context) error {
	names, err := conf.Servers(a.M, a.Conf)
	if err != nil {
		return err
	}
	for _, n := range names {
		file := conf.ServerPath(a.Conf, n)
		if file == a.Server {
			continue
		}
		text, err := a.M.ReadFile(file)
		if err != nil {
			return fmt.Errorf("%s: cannot read %s", stage.Answers, file)
		}
		if err := a.writeScratch(ctx, conf.ServerPath(a.Scratch, n), text); err != nil {
			return err
		}
	}
	return nil
}

var _ step.Step = (*Answers)(nil)
