package resource

import "context"

// Remark is a line that --check prints in the form of a change, which a run applies as nothing.
type Remark string

// String is the line that --check prints.
func (r Remark) String() string { return string(r) }

// Apply does nothing.
func (Remark) Apply(context.Context) error { return nil }
