package setup

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestMiseBinaryNeedsAPinForItsTarget(t *testing.T) {
	b := MiseBinary{M: effecttest.NewMachine(), Release: PinnedMise, Target: "linux-riscv64"}
	if _, err := b.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "linux-riscv64") {
		t.Errorf("err = %v", err)
	}
}

func TestMiseBinaryPlansNothingForTheVersion(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[MisePath+" --version"] = effecttest.Result{Out: []byte(PinnedMise.Version + " linux-x64 (2026-11-16)\n")}
	changes, err := MiseBinary{M: m, Release: PinnedMise, Target: "linux-x64"}.Plan(context.Background())
	if err != nil || len(changes) != 0 {
		t.Errorf("plan = %v, %v", changes, err)
	}
}

func TestMiseBinaryDownloadsThePinnedURL(t *testing.T) {
	body := []byte("mise")
	rel := MiseRelease{Version: "1.2.3", Sums: map[string]string{"linux-arm64": fmt.Sprintf("%x", sha256.Sum256(body))}}
	m := effecttest.NewMachine()
	m.Pages["https://github.com/jdx/mise/releases/download/v1.2.3/mise-v1.2.3-linux-arm64"] = body
	changes, err := MiseBinary{M: m, Release: rel, Target: "linux-arm64"}.Plan(context.Background())
	if err != nil || len(changes) != 1 {
		t.Fatalf("plan = %v, %v", changes, err)
	}
	if err := changes[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
	if string(m.Files[MisePath]) != "mise" || m.Modes[MisePath] != 0o755 {
		t.Errorf("mise = %q at mode %o", m.Files[MisePath], m.Modes[MisePath])
	}
}
