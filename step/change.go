package step

// ChangeKind says what a Change does to a machine, so that a screen tells changes apart. The
// values are those of --json.
type ChangeKind string

// The kinds of change.
const (
	// OtherChange is any change that doesn't say its kind.
	OtherChange ChangeKind = "other"
	Install     ChangeKind = "install"
	Write       ChangeKind = "write"
	Removal     ChangeKind = "removal"
	// Restart is any change to what a service is doing: a start, a stop or a reload too.
	Restart ChangeKind = "restart"
)

// Kinded is a Change that says its kind. It stands beside Change, as other modules build changes
// that have no kind.
type Kinded interface {
	Kind() ChangeKind
}

// Titled is a Change whose String has no line that names it, as a bare diff has none. It stands
// beside Change, as Kinded does, and leaves String as --check prints it.
type Titled interface {
	Title() string
}

// KindOf returns the change's kind.
func KindOf(c Change) ChangeKind {
	if k, ok := c.(Kinded); ok {
		return k.Kind()
	}
	return OtherChange
}
