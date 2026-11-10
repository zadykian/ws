package signoz

import (
	"embed"
	"encoding/json"
	"fmt"
	"path"
	"reflect"

	"github.com/zadykian/ws/effect"
)

// dashboardFiles holds the dashboards as SigNoz returns them, without the ID, owner and times it
// adds. Each file's name is the fixed name the step gives its dashboard.
//
//go:embed dashboards
var dashboardFiles embed.FS

// Dashboards is the step that keeps the dashboards of Names in SigNoz. Each is created from
// dashboards/NAME.json and replaced where it differs from that file. So a change made to it in
// SigNoz's UI lasts until the next run. A name taken off the list leaves its dashboard in SigNoz.
// The API answers only where Docker's daemon runs SigNoz, so without systemd booted the step skips.
type Dashboards struct {
	M     effect.Machine
	Out   effect.Out
	API   API
	Names []string
}

// Name returns the step's name.
func (Dashboards) Name() string { return "signoz dashboards" }

// WaitsOn returns the step that makes the API answer. The dashboards do not depend on the
// retention, so a check run reports both where both are due.
func (Dashboards) WaitsOn() string { return "signoz ready" }

// dashboard is a dashboard as SigNoz returns it.
type dashboard struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Locked bool            `json:"locked"`
	Spec   json.RawMessage `json:"spec"`
	Tags   json.RawMessage `json:"tags"`
	Image  json.RawMessage `json:"image"`
}

// differs reports whether a dashboard in SigNoz is not what the file holds.
func (d dashboard) differs(file dashboard) bool {
	return !sameJSON(d.Spec, file.Spec) || !sameJSON(d.Tags, file.Tags) || !sameJSON(d.Image, file.Image)
}

func sameJSON(a, b json.RawMessage) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

// load reads and checks the file of a dashboard. It names the dashboard as the list does, as two
// entries that found one dashboard would replace it in turn at every run.
func load(name string) (dashboard, []byte, error) {
	data, err := dashboardFiles.ReadFile(path.Join("dashboards", name+".json"))
	if err != nil {
		return dashboard{}, nil, fmt.Errorf("dashboard %s: %w", name, err)
	}
	var d dashboard
	if err := json.Unmarshal(data, &d); err != nil {
		return dashboard{}, nil, fmt.Errorf("dashboard %s: %w", name, err)
	}
	if d.Name != name {
		return dashboard{}, nil, fmt.Errorf("dashboards/%s.json names its dashboard %q, not %s", name, d.Name, name)
	}
	return d, data, nil
}
