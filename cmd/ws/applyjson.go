package main

import (
	"encoding/json"
	"io"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/server/apply"
)

// jsonEvents prints each event of a run as a line of JSON, and passes it on to next, which keeps
// ws's own lines on stderr. A write that fails stops the lines and stays in err.
type jsonEvents struct {
	enc  *json.Encoder
	next step.Observer
	err  error
}

func newJSONEvents(w io.Writer, next step.Observer) *jsonEvents {
	return &jsonEvents{enc: json.NewEncoder(w), next: next}
}

// Observe prints the event, then hands it on.
func (j *jsonEvents) Observe(e step.Event) {
	if j.err == nil {
		j.err = j.enc.Encode(apply.NewEventLine(e))
	}
	j.next.Observe(e)
}
