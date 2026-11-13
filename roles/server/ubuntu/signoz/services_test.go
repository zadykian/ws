package signoz

import (
	"slices"
	"testing"
)

func TestServicesAreTheOnesThatKeepRunning(t *testing.T) {
	got, err := Services(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("no service keeps running")
	}
	for _, oneShot := range []string{"signoz-telemetrystore-migrator", "signoz-telemetrystore-clickhouse-user-scripts"} {
		if slices.Contains(got, oneShot) {
			t.Errorf("%s runs once, but is listed", oneShot)
		}
	}
	if !slices.IsSorted(got) {
		t.Errorf("services %q are not sorted", got)
	}
}
