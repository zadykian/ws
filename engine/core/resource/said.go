package resource

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Said is a part whose changes say their headline to Out once they apply, as the Mac's do.
// Its changes carry no marker but Kind and Item.
type Said struct {
	Out  effect.Out
	Part Resource
}

// Plan plans the part's changes, each wrapped to say its headline, keeping its kind and its item.
// The wrapper forwards no marker, so Plan refuses a change that carries one: Forced, Warned, Early,
// Titled or Excerpted. The part's own error is kept.
func (s Said) Plan(ctx context.Context) ([]Change, error) {
	changes, err := s.Part.Plan(ctx)
	said := make([]Change, len(changes))
	for i, c := range changes {
		if marker := markerOf(c); marker != "" {
			return nil, fmt.Errorf("say %q: the change is %s, and a said change drops that", c, marker)
		}
		said[i] = speaking{s.Out, c}
		if _, ok := c.(step.Itemized); ok {
			said[i] = saidItem{speaking{s.Out, c}}
		}
	}
	return said, err
}

// markerOf names the marker that c carries, or "" where it has none.
func markerOf(c Change) string {
	switch c.(type) {
	case Forced:
		return "Forced"
	case Warned:
		return "Warned"
	case Early:
		return "Early"
	case step.Titled:
		return "Titled"
	case step.Excerpted:
		return "Excerpted"
	}
	return ""
}

// Report is the part's, where it reports.
func (s Said) Report(ctx context.Context) ([]step.Item, error) {
	if r, ok := s.Part.(step.Reporter); ok {
		return r.Report(ctx)
	}
	return nil, nil
}

type speaking struct {
	out effect.Out
	Change
}

func (s speaking) Kind() step.ChangeKind { return step.KindOf(s.Change) }

func (s speaking) Apply(ctx context.Context) error {
	if err := s.Change.Apply(ctx); err != nil {
		return err
	}
	s.out.Say("%s", step.Headline(s.Change))
	return nil
}

type saidItem struct{ speaking }

func (s saidItem) Item() step.Item { return s.Change.(step.Itemized).Item() }
