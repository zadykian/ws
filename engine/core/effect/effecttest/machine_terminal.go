package effecttest

import "context"

// RunTerminal records "terminal NAME ARGS", apart from Run's, as only this run may ask on the
// terminal. It returns its result from the table.
func (m *Machine) RunTerminal(_ context.Context, name string, args ...string) ([]byte, error) {
	return m.run("terminal", name, args)
}
