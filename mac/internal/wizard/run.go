package wizard

import (
	"context"
	"strconv"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// Run runs the stages in order under the run's lock, with the Mac kept from idle sleep until it
// ends: a stage may wait on a download or an answer. pid is the process caffeinate waits on. It
// reports whether any stage had changes to make, as step.Run does.
func Run(ctx context.Context, m effect.Machine, out effect.Out, stages []step.Step, check bool, tmp string, pid int) (bool, error) {
	lock, release, err := Lock(ctx, m, tmp)
	if err != nil {
		return false, err
	}
	defer release()
	// The masters that the stages' ssh calls opened end before the lock goes, after a signal too.
	defer SSH{M: m, Lock: lock}.End(context.WithoutCancel(ctx))
	awake, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		// The command ends with the cancel, or with the process it waits on.
		_, _ = m.Run(awake, "caffeinate", "-i", "-w", strconv.Itoa(pid))
	}()
	defer func() {
		cancel()
		<-done
	}()
	return step.Run(ctx, stages, out, check)
}
