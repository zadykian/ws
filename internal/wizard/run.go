package wizard

import (
	"context"
	"github.com/zadykian/ws/engine/core/remote"
	"strconv"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Run runs the stages in order under the run's lock. It keeps the Mac from idle sleep until it
// ends, as a stage may wait on a download or an answer. pid is the process caffeinate waits on, and
// 0 keeps nothing awake, as on ws, where caffeinate is not macOS's. It reports whether any stage
// had changes to make, as step.Run does. closing runs after a run that changed something, outside
// --check, while the lock stands: the stages' files in it are still there. o is the stages' run:
// who hears it, and where a screen stops it or leaves a stage out.
func Run(ctx context.Context, m effect.Machine, o step.Options, stages []step.Step, tmp string, pid int, closing func(context.Context)) (bool, error) {
	lock, release, err := Lock(ctx, m, tmp)
	if err != nil {
		return false, err
	}
	defer release()
	// The masters that the stages' ssh calls opened end before the lock goes, after a signal too.
	defer remote.SSH{M: m, Lock: lock}.End(context.WithoutCancel(ctx))
	awake, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		// The command ends with the cancel, or with the process it waits on.
		if pid != 0 {
			_, _ = m.Run(awake, "caffeinate", "-i", "-w", strconv.Itoa(pid))
		}
	}()
	defer func() {
		cancel()
		<-done
	}()
	changed, err := step.RunWith(ctx, stages, o)
	if err == nil && changed && !o.Check && closing != nil {
		closing(ctx)
	}
	return changed, err
}
