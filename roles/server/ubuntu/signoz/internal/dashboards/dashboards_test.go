package dashboards

import "testing"

func TestEachDashboardFileNamesItsDashboard(t *testing.T) {
	for _, n := range []string{"host", "claude"} {
		if _, _, err := load(n); err != nil {
			t.Error(err)
		}
	}
	if _, _, err := load("missing"); err == nil {
		t.Error("load(missing) = nil, want an error")
	}
}
