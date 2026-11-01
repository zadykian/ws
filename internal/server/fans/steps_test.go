package fans

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/values"
)

const (
	vendor = "Micro-Star International Co., Ltd."
	kernel = "7.0.0-38-generic"
)

// onTheBoard returns a machine whose board is the vendor's, running the kernel.
func onTheBoard() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Files[boardVendorFile] = []byte(vendor + "\n")
	m.Results["uname -r"] = effecttest.Result{Out: []byte(kernel + "\n")}
	return m
}

func TestStepsAreInOrderEachOnTheOneBefore(t *testing.T) {
	var names, after []string
	for _, s := range steps(values.Default().Fans, onTheBoard(), &effecttest.Out{}) {
		names, after = append(names, s.Name()), append(after, s.WaitsOn())
	}
	if want := []string{boardStep, packagesStep, sourceStep, driverStep, modulesStep}; !slices.Equal(names, want) {
		t.Errorf("steps = %q, want %q", names, want)
	}
	if want := []string{"", boardStep, packagesStep, sourceStep, driverStep}; !slices.Equal(after, want) {
		t.Errorf("after = %q, want %q", after, want)
	}
}

func TestStepsPlanNothingOnAnotherBoard(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Files[boardVendorFile] = []byte("QEMU\n")
	for _, s := range steps(values.Default().Fans, m, out) {
		p, err := s.Plan(t.Context())
		if err != nil || len(p.Changes) != 0 {
			t.Errorf("%s: Plan = %v, %v; want none", s.Name(), p, err)
		}
	}
	if !strings.Contains(strings.Join(out.Lines, "\n"), "leaves its fans as they are") {
		t.Errorf("says %q", out.Lines)
	}
}

func TestPackagesNameTheRunningKernelsHeaders(t *testing.T) {
	m := onTheBoard()
	m.Results["dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n dkms linux-headers-generic linux-headers-"+kernel] = effecttest.Result{}
	p, err := steps(values.Default().Fans, m, &effecttest.Out{})[1].Plan(t.Context())
	if err != nil || len(p.Changes) == 0 || !strings.Contains(p.Changes[0].String(), "linux-headers-"+kernel) {
		t.Errorf("Plan = %v, %v; want an install naming the kernel's headers", p, err)
	}
}

func TestDriverPlansWhatDKMSLacks(t *testing.T) {
	const status = "nct6687d/old, " + kernel + ", x86_64: installed\n"
	for _, tc := range []struct {
		name, status string
		noDKMS       bool
		want         []string
	}{
		{"nothing yet", "", true, []string{"add the driver to DKMS", "build and install the driver for " + kernel}},
		{"added", "nct6687d/" + nct6687Short + ": added\n", false, []string{"build and install the driver for " + kernel}},
		{"installed", "nct6687d/" + nct6687Short + ", " + kernel + ", x86_64: installed\n", false, nil},
		{"another version", status, false, []string{"remove the driver's version old from DKMS", "remove the source of version old", "add the driver to DKMS", "build and install the driver for " + kernel}},
	} {
		m := onTheBoard()
		m.Results["dkms status nct6687d"] = effecttest.Result{Out: []byte(tc.status)}
		if !tc.noDKMS {
			m.Files[dkmsBinary] = []byte("x")
		}
		changes, err := driver{m}.Plan(t.Context())
		var got []string
		for _, c := range changes {
			got = append(got, c.String())
		}
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("%s: Plan = %q, %v; want %q", tc.name, got, err, tc.want)
		}
	}
}
