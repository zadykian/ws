package state

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/tui/internal/run"
)

// Reader reads a machine and changes nothing. b gives the read's effects, for its lines to
// reach the screen and not the terminal under it.
type Reader func(ctx context.Context, b *run.Bridge) ([]run.StateSection, error)

// RunState draws the state screen on c.Out until q, and reads the machine in Run's work. So a
// read's panic and a signal end as a run's do. A read goes on under the screen, which takes the
// keys meanwhile, and q ends it through ctx. It returns nil after q, and Cut's error after a cut.
func RunState(ctx context.Context, c run.Config, title string, read Reader) error {
	ctx, end := context.WithCancelCause(ctx)
	defer end(nil)
	// The screen asks once a read has ended, so serve never has two reads to run.
	a := run.NewAsks(end)
	ask := func() tea.Msg {
		a.Ask(func(ctx context.Context, b *run.Bridge) tea.Msg {
			sections, err := read(ctx, b)
			return Read{sections, err}
		})
		return nil
	}
	screen := NewScreen(c.Theme, title, ask, a.Leave)
	return run.Run(ctx, c, screen, a.Serve)
}
