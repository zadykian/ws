package engine

import (
	"encoding/json"
	"io"

	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
)

// jsonEvents prints each event of a run as a line of JSON, and passes it on to next, which keeps
// ws's own lines on stderr. A write that fails stops the lines and stays in err.
type jsonEvents struct {
	enc  *json.Encoder
	next step.Observer
	err  error
}

// Err returns the first write error, which stopped the lines.
func (j *jsonEvents) Err() error { return j.err }

// newJSONEvents returns events that print to w and pass each one on to next.
func newJSONEvents(w io.Writer, next step.Observer) *jsonEvents {
	return &jsonEvents{enc: json.NewEncoder(w), next: next}
}

// Observe prints the event, then hands it on.
func (j *jsonEvents) Observe(e step.Event) {
	if j.err == nil {
		j.err = j.enc.Encode(role.NewEventLine(e))
	}
	j.next.Observe(e)
}
