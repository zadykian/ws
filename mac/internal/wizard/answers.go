package wizard

import (
	"fmt"
	"strings"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
)

// Answers is the stage that gives mac.conf the values it lacks, each from the environment, under
// its key's name, or asked at the terminal. It runs before any stage that reads mac.conf.
type Answers struct {
	// Conf is mac.conf's path, and Roles the shell role's defaults, whose git identity ws's git takes.
	Conf, Roles string
	// Prog prefixes the messages of a refused answer.
	Prog string
	M    effect.Machine
	P    effect.Prompt
	Out  effect.Out
	// Env looks a variable up, telling one set to nothing from one that is not set.
	Env func(string) (string, bool)
}

// Name is the stage's name.
func (*Answers) Name() string { return answers }

// WaitsOn is empty: nothing is set up before mac.conf.
func (*Answers) WaitsOn() string { return "" }

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
