package wizard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

func TestPkgSigner(t *testing.T) {
	good := "Package \"Homebrew.pkg\":\n   Status: signed by a developer certificate issued by Apple for distribution\n" +
		"   Notarization: trusted by the Apple notary service\n   Certificate Chain:\n" +
		"    1. " + brewSigner + "\n       Expires: 2030\n    2. Developer ID Certification Authority\n"
	for name, tc := range map[string]struct{ out, want string }{
		"issued by Apple": {good, brewSigner},
		"no status":       {"   1. Mallory\n", ""},
		"before status": {"    1. Mallory\n   Status: signed by a developer certificate issued by Apple for distribution\n" +
			"    1. " + brewSigner + "\n", brewSigner},
		"unsigned": {"Package \"x.pkg\":\n   Status: no signature\n", ""},
		"empty":    {"", ""},
	} {
		if got := pkgSigner(tc.out); got != tc.want {
			t.Errorf("%s: pkgSigner = %q, want %q", name, got, tc.want)
		}
	}
}

func TestBrewSHA256IsHex(t *testing.T) {
	if b, err := hex.DecodeString(brewSHA256); err != nil || len(b) != sha256.Size {
		t.Errorf("brewSHA256 = %q, want %d hex bytes", brewSHA256, sha256.Size)
	}
}

func TestHomebrewWaitsOnTools(t *testing.T) {
	if got := (&Homebrew{}).WaitsOn(); got != toolsStage {
		t.Errorf("WaitsOn = %q, want %q", got, toolsStage)
	}
	for _, s := range []step.Step{&Answers{}, &Gh{}, &Setup{}, &Cable{}} {
		if got := s.WaitsOn(); got != homebrewStage {
			t.Errorf("%s waits on %q, want %q", s.Name(), got, homebrewStage)
		}
	}
}

func TestHomebrewPlan(t *testing.T) {
	m := effecttest.NewMachine()
	h := &Homebrew{Lock: "/t/lock", M: m, Out: &effecttest.Out{}}
	m.Results[brewPath+" --version"] = effecttest.Result{Err: errors.New("none")}
	if p, err := h.Plan(context.Background()); err != nil || len(p.Changes) != 1 {
		t.Errorf("Plan without brew = %v, %v", p.Changes, err)
	}
	m.Results[brewPath+" --version"] = effecttest.Result{}
	if p, err := h.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("Plan with brew = %v, %v", p.Changes, err)
	}
}

// newBrew returns the stage with a download that curl leaves as pkg, signed by signer.
func newBrew(pkg []byte, signer string) (*Homebrew, *effecttest.Machine) {
	m := effecttest.NewMachine()
	file := "/t/lock/Homebrew.pkg"
	m.Files[file] = pkg
	m.Results["curl -fsSL -o "+file+" "+brewURL] = effecttest.Result{}
	m.Results["pkgutil --check-signature "+file] = effecttest.Result{Out: []byte(
		"   Status: signed by a developer certificate issued by Apple for distribution\n    1. " + signer + "\n")}
	m.Results["rm -f "+file] = effecttest.Result{}
	m.Results["installer -pkg "+file+" -target /"] = effecttest.Result{}
	m.Results[brewPath+" --version"] = effecttest.Result{}
	return &Homebrew{Lock: "/t/lock", M: m, Out: &effecttest.Out{}}, m
}

func TestHomebrewApplyRefuses(t *testing.T) {
	sum := sha256.Sum256([]byte("pkg"))
	for name, tc := range map[string]struct {
		pkg    string
		signer string
		want   string
	}{
		"cut pkg":      {"cut", brewSigner, "SHA-256 is"},
		"other signer": {"pkg", "Developer ID Installer: Jane Doe (ABCDE12345)", "is signed by 'Developer ID Installer: Jane Doe"},
	} {
		h, m := newBrew([]byte(tc.pkg), tc.signer)
		h.Sum = hex.EncodeToString(sum[:])
		err := h.Apply(context.Background(), step.Plan{})
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: Apply = %v, want %q", name, err, tc.want)
		}
		if slices.Contains(m.Calls, "root installer -pkg /t/lock/Homebrew.pkg -target /") {
			t.Errorf("%s: installer ran on a refused pkg", name)
		}
		if !slices.Contains(m.Calls, "run rm -f /t/lock/Homebrew.pkg") {
			t.Errorf("%s: the pkg stays: %v", name, m.Calls)
		}
	}
}

func TestHomebrewApplyInstalls(t *testing.T) {
	sum := sha256.Sum256([]byte("pkg"))
	h, m := newBrew([]byte("pkg"), brewSigner)
	h.Sum = hex.EncodeToString(sum[:])
	if err := h.Apply(context.Background(), step.Plan{}); err != nil {
		t.Fatalf("Apply = %v", err)
	}
	want := []string{"run curl -fsSL -o /t/lock/Homebrew.pkg " + brewURL, "root installer -pkg /t/lock/Homebrew.pkg -target /"}
	for _, w := range want {
		if !slices.Contains(m.Calls, w) {
			t.Errorf("calls = %v, want %q", m.Calls, w)
		}
	}
}
