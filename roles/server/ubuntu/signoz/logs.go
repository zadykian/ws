package signoz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// logsTTL is the retention of logs, with the rules that SigNoz's UI sets to keep some for another
// time.
type logsTTL struct {
	Days       int             `json:"default_ttl_days"`
	Status     string          `json:"status"`
	Conditions json.RawMessage `json:"ttl_conditions"`
}

func (l logsTTL) String() string { return fmt.Sprintf("logs: %d days, %s", l.Days, l.Status) }

func (r Retention) readLogs(ctx context.Context) (logsTTL, error) {
	var l logsTTL
	err := r.API.Get(ctx, "/api/v2/settings/ttl", &l)
	return l, err
}

// setLogs sets the retention of logs.
type setLogs struct {
	r    Retention
	have logsTTL
}

func (c setLogs) String() string {
	return fmt.Sprintf("set the retention of logs to %d days (now %s)", c.r.Days, c.have)
}

// Apply sets the retention of logs. SigNoz alters the tables before it answers. The rules set in
// its UI are sent back as they were, so they stay.
func (c setLogs) Apply(ctx context.Context) error {
	conditions := c.have.Conditions
	if len(conditions) == 0 {
		conditions = json.RawMessage("[]")
	}
	body, err := json.Marshal(map[string]any{
		"type": "logs", "defaultTTLDays": c.r.Days, "ttlConditions": conditions,
	})
	if err != nil {
		return err
	}
	return c.r.API.Call(ctx, http.MethodPost, "/api/v2/settings/ttl", body, http.StatusOK, nil)
}

// due says whether the retention differs from the wanted days, or the last change is not done.
func (l logsTTL) due(days int) bool { return l.Days != days || l.Status != "success" }
