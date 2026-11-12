package setup

import "context"

// Finish hands the tests of package setup_test the finish of a run.
func (t *Tunnel) Finish(ctx context.Context) error { return t.finish(ctx) }
