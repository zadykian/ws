package tunnel

import (
	"reflect"
	"testing"
)

func TestPlanRoutes(t *testing.T) {
	tests := []struct {
		name         string
		want, routed []string
		delete, add  []string
	}{
		{"nothing in place", []string{"10.1.0.0/16", "10.2.0.0/16"}, nil, nil, []string{"10.1.0.0/16", "10.2.0.0/16"}},
		{"all in place", []string{"10.1.0.0/16"}, []string{"10.1.0.0/16"}, nil, nil},
		{"one goes and one comes", []string{"10.2.0.0/16"}, []string{"10.1.0.0/16"}, []string{"10.1.0.0/16"}, []string{"10.2.0.0/16"}},
		{"all go", nil, []string{"10.1.0.0/16", "10.2.0.0/16"}, []string{"10.1.0.0/16", "10.2.0.0/16"}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := PlanRoutes(nets(tc.want...), nets(tc.routed...))
			if !reflect.DeepEqual(got.Delete, nets(tc.delete...)) || !reflect.DeepEqual(got.Add, nets(tc.add...)) {
				t.Errorf("delete %v, add %v; want %v, %v", got.Delete, got.Add, tc.delete, tc.add)
			}
		})
	}
}

func TestApplied(t *testing.T) {
	plan := PlanRoutes(nets("10.1.0.0/16", "10.2.0.0/16", "10.3.0.0/16"), nets("10.1.0.0/16"))
	tests := []struct {
		name                    string
		failed, out, unroutable []string
		routed, unroutableNow   []string
		logs                    int
	}{
		{"all routed", nil, nil, nil, []string{"10.1.0.0/16", "10.2.0.0/16", "10.3.0.0/16"}, nil, 0},
		{"one fails and is logged", []string{"10.2.0.0/16"}, nil, nil,
			[]string{"10.1.0.0/16", "10.3.0.0/16"}, []string{"10.2.0.0/16"}, 1},
		{"a failure again is not logged", []string{"10.2.0.0/16"}, nil, []string{"10.2.0.0/16"},
			[]string{"10.1.0.0/16", "10.3.0.0/16"}, []string{"10.2.0.0/16"}, 0},
		{"a network left out keeps its mark", nil, []string{"10.9.0.0/16", "10.8.0.0/16"}, []string{"10.9.0.0/16"},
			[]string{"10.1.0.0/16", "10.2.0.0/16", "10.3.0.0/16"}, []string{"10.9.0.0/16"}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			routed, unroutable, logs := plan.Applied(nets(tc.failed...), nets(tc.out...), nets(tc.unroutable...), "utun4")
			if !reflect.DeepEqual(routed, nets(tc.routed...)) || !reflect.DeepEqual(unroutable, nets(tc.unroutableNow...)) || len(logs) != tc.logs {
				t.Errorf("routed %v, unroutable %v, logs %q", routed, unroutable, logs)
			}
		})
	}
}

func TestAppliedLog(t *testing.T) {
	plan := PlanRoutes(nets("10.2.0.0/16"), nil)
	_, _, logs := plan.Applied(nets("10.2.0.0/16"), nil, nil, "utun4")
	want := []string{"cannot route 10.2.0.0/16 into utun4, as another route to it may be in the way"}
	if !reflect.DeepEqual(logs, want) {
		t.Errorf("logs %q, want %q", logs, want)
	}
}
