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

// Excerpted is a Change whose String quotes a file of the user's, which may hold a secret. --check
// prints that on the user's own terminal, and a report and a run's log outlast it, so they hold the
// Excerpt. It stands beside Change, as Titled does, and leaves String as --check prints it.
type Excerpted interface {
	Excerpt() string
}

// ExcerptOf returns the change as a report and a log hold it.
func ExcerptOf(c Change) string {
	if e, ok := c.(Excerpted); ok {
		return e.Excerpt()
	}
	return c.String()
}

// KindOf returns the change's kind.
func KindOf(c Change) ChangeKind {
	if k, ok := c.(Kinded); ok {
		return k.Kind()
	}
	return OtherChange
}
