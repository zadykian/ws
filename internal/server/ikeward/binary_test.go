package ikeward

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func planDownload(t *testing.T, arch string) ([]host.Change, error) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte(arch + "\n")}
	parts, err := download(m)(t.Context())
	if err != nil {
		return nil, err
	}
	return parts[0].Plan(t.Context())
}

func TestDownloadFetchesThePinnedBinaryForTheArchitecture(t *testing.T) {
	changes, err := planDownload(t, "arm64")
	if err != nil || len(changes) != 1 {
		t.Fatalf("plan = %v, %v; want one change", changes, err)
	}
	want := "download " + releases + "/download/v" + version + "/ikeward-linux-arm64 to " + binary
	if got := changes[0].String(); got != want {
		t.Errorf("change = %q, want %q", got, want)
	}
}

func TestDownloadRefusesAnArchitectureWithNoRelease(t *testing.T) {
	if _, err := planDownload(t, "riscv64"); err == nil || !strings.Contains(err.Error(), "riscv64") {
		t.Errorf("error = %v, want one naming riscv64", err)
	}
}
