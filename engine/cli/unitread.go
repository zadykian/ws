package cli

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
)

// unitRead returns ws unit read.
func unitRead(p Env, ss sides) *cobra.Command {
	c := &cobra.Command{
		Use:   "read UNIT...",
		Short: "print each unit's state: " + InSync + ", or the first change that write would make",
		Long: "It plans each UNIT and the units it waits on, and applies nothing. It prints a line per UNIT " +
			"on stdout, in the order given: the unit, a tab and its state. The state is " + InSync + "; the " +
			"headline of its first change, with \" (+N more)\" for the rest; \"after UNIT\" where it waits " +
			"on a unit with changes; or \"fails: \" and why its plan failed. A failed plan stops no other " +
			"unit's read. --whole takes the UNITs for every line of mise's plan, as the plugin passes them: " +
			"where the specs hold a unit they lack that is not " + InSync + ", each UNIT fails and names it, " +
			"and each fails where ws runs outside the side's run directory, which mise must run in. " +
			"ws's own lines go to stderr.",
		Args: cobra.MinimumNArgs(1),
	}
	whole := c.Flags().Bool("whole", false, "fail each UNIT where they lack a unit of the specs with a change")
	c.RunE = func(c *cobra.Command, names []string) error {
		s, err := ss.pick(p.Pwd)
		if err != nil && *whole {
			// mise runs in no side's run directory, which loads none of the specs' lines.
			return failEach(c, names, err)
		}
		if err != nil {
			return err
		}
		if err := s.refuse(p); err != nil {
			return err
		}
		r, err := loadUnits(c, p, s)
		defer r.term.Flush()
		if err != nil {
			return err
		}
		if why := r.outsideRun(p.Pwd); *whole && why != "" {
			// Such a run would skip every role, so the read fails each line, in sync or not.
			var sb strings.Builder
			for _, name := range names {
				sb.WriteString(name + "\tfails: " + why + "\n")
			}
			_, err = fmt.Fprint(c.OutOrStdout(), sb.String())
			return err
		}
		missing := lacking(*whole, r.all, names)
		units, err := r.closure(append(slices.Clone(names), missing...))
		if err != nil {
			return err
		}
		reports, err := step.Read(c.Context(), kind.Steps(units))
		if err != nil {
			return err
		}
		state := map[string]string{}
		for _, rep := range reports {
			state[rep.Step] = unitState(rep)
		}
		var due []string
		for _, u := range missing {
			if state[u] != InSync {
				due = append(due, manager+":"+u)
			}
		}
		if len(due) > 0 {
			for _, name := range names {
				state[name] = "fails: mise's plan lacks " + strings.Join(due, ", ") +
					": add each to [bootstrap.packages], or run mise in " + r.runDir()
			}
		}
		var sb strings.Builder
		for _, name := range names {
			sb.WriteString(name + "\t" + state[name] + "\n")
		}
		_, err = fmt.Fprint(c.OutOrStdout(), sb.String())
		return err
	}
	return c
}

// failEach prints each unit as failing with why, which is a read that cannot load any unit.
func failEach(c *cobra.Command, names []string, why error) error {
	var sb strings.Builder
	for _, name := range names {
		sb.WriteString(name + "\tfails: " + why.Error() + "\n")
	}
	_, err := fmt.Fprint(c.OutOrStdout(), sb.String())
	return err
}

// sameDir reports whether a and b name one directory, through any link. A path that does not
// resolve is compared as written.
func sameDir(a, b string) bool {
	resolve := func(p string) string {
		if r, err := filepath.EvalSymlinks(p); err == nil {
			return r
		}
		return filepath.Clean(p)
	}
	return resolve(a) == resolve(b)
}

// lacking returns each unit of all that names lacks where whole says that names are every line of
// mise's plan. mise plans the lines alone, so a line gone from a role's file is a unit that no run
// sets up. The plugin's read hook passes whole, so that every plan and run fails on it, with no
// call of mise from within mise's own read. mise reads the lines once more before it installs,
// with those in sync left out, so the read fails on a lacking unit with a change alone.
func lacking(whole bool, all []kind.Unit, names []string) []string {
	if !whole {
		return nil
	}
	var missing []string
	for _, u := range kind.Names(all) {
		if !slices.Contains(names, u) {
			missing = append(missing, u)
		}
	}
	return missing
}

// unitState is a unit's report as one line: what it waits on, why its plan failed, the first
// change it would make, or InSync. A failure goes before the changes, as mise must fail the plan.
func unitState(r step.Report) string {
	if r.After != "" {
		return "after " + r.After
	}
	for _, it := range r.Items {
		if it.Kind == step.CheckItem {
			why := strings.TrimPrefix(it.Conflict, r.Step+": planning: ")
			line, _, _ := strings.Cut(why, "\n")
			return "fails: " + line
		}
	}
	if len(r.Changes) == 0 {
		return InSync
	}
	line, _, _ := strings.Cut(r.Changes[0].Text, "\n")
	if more := len(r.Changes) - 1; more > 0 {
		line += fmt.Sprintf(" (+%d more)", more)
	}
	return line
}

// unitWrite returns ws unit write.
func unitWrite(p Env, ss sides) *cobra.Command {
	c := &cobra.Command{
		Use:   "write UNIT...",
		Short: "apply the units, in the order they run",
		Long: "It plans, applies and plans again each UNIT, in the order the units run whatever the order " +
			"given. A unit that a UNIT waits on and that is not named is planned and left out. A UNIT " +
			"behind one with changes is skipped, and fails the write. --dry-run applies nothing and " +
			"prints each change, file diffs included. ws's lines go to stderr, and the last one names " +
			"the units that changed.",
		Args: cobra.MinimumNArgs(1),
	}
	dry := c.Flags().Bool("dry-run", false, "plan the units and apply none")
	c.RunE = func(c *cobra.Command, names []string) error {
		s, err := ss.pick(p.Pwd)
		if err != nil {
			return err
		}
		if err := s.refuse(p); err != nil {
			return err
		}
		r, err := loadUnits(c, p, s)
		defer r.term.Flush()
		if err != nil {
			return err
		}
		units, err := r.closure(names)
		if err != nil {
			return err
		}
		w := &written{Observer: step.Printer{Out: r.out}, named: map[string]bool{}}
		for _, n := range names {
			w.named[n] = true
		}
		_, err = step.RunWith(c.Context(), kind.Steps(units), step.Options{Observer: w, Check: *dry,
			LeftOut: func(name string) bool { return !w.named[name] }})
		if err != nil {
			return err
		}
		verb, done := "changed", "none"
		if *dry {
			verb = "would change"
		}
		if len(w.changed) > 0 {
			done = strings.Join(w.changed, ", ")
		}
		r.out.Say("%s: %s", verb, done)
		if len(w.behind) > 0 && !*dry {
			return fmt.Errorf("skipped %s: write the unit it waits on too", strings.Join(w.behind, ", "))
		}
		return nil
	}
	return c
}

// written follows a write: the named units that changed, and those that a unit left out kept
// back. A tool that called the write would count those done, so they fail it.
type written struct {
	step.Observer
	named   map[string]bool
	changed []string
	behind  []string
}

func (w *written) Observe(e step.Event) {
	switch {
	case e.Kind == step.Done && !e.LeftOut && len(e.Changes) > 0:
		w.changed = append(w.changed, e.Step)
	case e.Kind == step.Skipped && e.After != "" && w.named[e.Step]:
		w.behind = append(w.behind, e.Step+" (after "+e.After+")")
	}
	w.Observer.Observe(e)
}
