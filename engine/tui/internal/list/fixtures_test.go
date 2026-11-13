package list

import (
	"github.com/zadykian/ws/engine/tui/internal/tuitest"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// The keys and fixtures that the tests share, under the names these tests use.
var (
	stripped = tuitest.Stripped
	planned  = tuitest.Planned

	testTheme = widget.NewTheme(func(string) string { return "" })
)

type oneChange = tuitest.OneChange
