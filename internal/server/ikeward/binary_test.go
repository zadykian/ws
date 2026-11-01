package ikeward

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect"
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

// flaky fails its first downloads.
type flaky struct {
	*effecttest.Machine
	fail, tried int
}

func (f *flaky) Download(ctx context.Context, url, path, sum string, a effect.Attrs) error {
	f.tried++
	if f.tried <= f.fail {
		return errors.New("connection reset")
	}
	return f.Machine.Download(ctx, url, path, sum, a)
}

func TestDownloadTriesAgainAfterAFailure(t *testing.T) {
	wait := downloadWait
	downloadWait = 0
	t.Cleanup(func() { downloadWait = wait })
	for fail, wantErr := range map[int]bool{downloadTries - 1: false, downloadTries: true} {
		m := &flaky{Machine: effecttest.NewMachine(), fail: fail}
		url := releases + "/download/v" + version + "/ikeward-linux-amd64"
		body := []byte("binary")
		m.Pages[url] = body
		pin := sums["amd64"]
		sums["amd64"] = fmt.Sprintf("%x", sha256.Sum256(body))
		t.Cleanup(func() { sums["amd64"] = pin })
		m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
		parts, err := download(m)(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		changes, err := parts[0].Plan(t.Context())
		if err != nil || len(changes) != 1 {
			t.Fatalf("plan = %v, %v", changes, err)
		}
		if err := changes[0].Apply(t.Context()); (err != nil) != wantErr {
			t.Errorf("%d failures: error = %v, want an error: %v", fail, err, wantErr)
		}
	}
}
