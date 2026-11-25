package retention

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ttl is what SigNoz says of the retention of traces or metrics.
type ttl struct {
	signal string
	hours  int
	status string
}

func (t ttl) String() string { return fmt.Sprintf("%s: %d h, %s", t.signal, t.hours, t.status) }

// readSignal reads the retention of traces or metrics.
func (r Retention) readSignal(ctx context.Context, signal string) (ttl, error) {
	var got map[string]json.RawMessage
	if err := r.API.Get(ctx, "/api/v1/settings/ttl?type="+signal, &got); err != nil {
		return ttl{}, err
	}
	t := ttl{signal: signal}
	if err := json.Unmarshal(got[signal+"_ttl_duration_hrs"], &t.hours); err != nil {
		return ttl{}, fmt.Errorf("%s: its retention in hours: %w", signal, err)
	}
	if err := json.Unmarshal(got["status"], &t.status); err != nil {
		return ttl{}, fmt.Errorf("%s: its status: %w", signal, err)
	}
	return t, nil
}

// setSignals sets the retention of the signals that are due.
type setSignals struct {
	r   Retention
	due []ttl
}

func (c setSignals) String() string {
	parts := make([]string, len(c.due))
	for i, t := range c.due {
		parts[i] = fmt.Sprintf("%s (now %s)", t.signal, t)
	}
	return fmt.Sprintf("set the retention of %s to %d h", strings.Join(parts, " and "), c.r.Days*24)
}

// Apply sets the retention of each signal, then waits for SigNoz to alter the tables, as it
// answers at once and alters them after.
func (c setSignals) Apply(ctx context.Context) error {
	hours := c.r.Days * 24
	for _, t := range c.due {
		q := fmt.Sprintf("/api/v1/settings/ttl?type=%s&duration=%dh", t.signal, hours)
		if err := c.r.API.Call(ctx, http.MethodPost, q, nil, http.StatusOK, nil); err != nil {
			return err
		}
	}
	w := c.r.Altered.OrDefault(60, 5*time.Second)
	for _, t := range c.due {
		err := w.Poll(ctx, t.signal+"'s retention", func() (bool, string, error) {
			got, err := c.r.readSignal(ctx, t.signal)
			if err != nil {
				return false, err.Error(), nil
			}
			return got.status == "success" && got.hours == hours, got.String(), nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// due says whether the retention differs from the wanted hours, or its last change failed.
func (t ttl) due(hours int) bool { return t.hours != hours || t.status == "failed" }
