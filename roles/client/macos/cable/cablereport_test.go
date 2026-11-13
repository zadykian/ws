package cable_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
)

func TestCableReportsItsAddress(t *testing.T) {
	const list = "header\nDock\n"
	m := effecttest.NewMachine()
	it := mactest.ReportOf(t, cableStep(m, &effecttest.Out{}, list, cableSet).Step())["Dock"]
	if it.Kind != step.ServiceItem || it.Found != it.Want {
		t.Errorf("set = %+v, want its address found", it)
	}
	it = mactest.ReportOf(t, cableStep(m, &effecttest.Out{}, list, "DHCP Configuration\nIP address: 10.0.0.9\n").Step())["Dock"]
	if it.Found == it.Want || !strings.Contains(it.Found, "10.0.0.9") {
		t.Errorf("dhcp = %+v, want the address it has", it)
	}
	it = mactest.ReportOf(t, cableStep(m, &effecttest.Out{}, "header\n*Dock\n", cableSet).Step())["Dock"]
	if it.Found == it.Want || !strings.Contains(it.Conflict, "disabled") {
		t.Errorf("disabled = %+v, want a conflict: no run turns it on", it)
	}
	s := cableStep(m, &effecttest.Out{}, list, cableSet)
	s.Conf = conf.Conf{}
	if got := mactest.ReportOf(t, s.Step()); len(got) != 0 {
		t.Errorf("items = %v, want none without a service", got)
	}
}

// A report warns of nothing: a plan says the warnings, and a report only reads.
func TestCableReportWarnsNothing(t *testing.T) {
	out := &effecttest.Out{}
	_, err := cableStep(effecttest.NewMachine(), out, "header\n*Dock\n", cableSet).Step().Report(context.Background())
	if err != nil || len(out.Lines) != 0 {
		t.Errorf("Report = %v, said %q; want no warning", err, out.Lines)
	}
}
