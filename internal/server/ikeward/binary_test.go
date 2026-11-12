package ikeward

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func planDownload(t *testing.T, arch string) ([]host.Change, error) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte(arch + "\n")}
	return binaryFile(m).Plan(t.Context())
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
