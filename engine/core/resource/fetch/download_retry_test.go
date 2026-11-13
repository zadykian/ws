package fetch_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/fetch"
)

// flaky fails the first fails Downloads with err, and counts them all.
type flaky struct {
	*effecttest.Machine
	fails, calls int
	err          error
}

func (f *flaky) Download(ctx context.Context, url, path, sum string, a effect.Attrs) error {
	f.calls++
	if f.calls <= f.fails {
		return f.err
	}
	return f.Machine.Download(ctx, url, path, sum, a)
}

func fetchOnce(ctx context.Context, t *testing.T, m effect.Machine, sum string) error {
	changes, err := fetch.Download{M: m, URL: "u", Path: "/f", SHA256: sum, Mode: 0o644}.Plan(ctx)
	if err != nil || len(changes) != 1 {
		t.Fatalf("plan = %v, %v; want one change", changes, err)
	}
	return changes[0].Apply(ctx)
}

const bodySum = "230d8358dc8e8890b4c58deeb62912ee2f20357ae92a5cc861b98e68fe31acb5" // SHA-256 of "body"

func TestDownloadRetriesAFailedFetch(t *testing.T) {
	defer fetch.FixRetryWait(time.Millisecond)()
	m := effecttest.NewMachine()
	m.Pages["u"] = []byte("body")
	f := &flaky{Machine: m, fails: 2, err: errors.New("503")}
	if err := fetchOnce(t.Context(), t, f, bodySum); err != nil || f.calls != 3 {
		t.Errorf("err = %v after %d calls, want none after 3", err, f.calls)
	}
}

func TestDownloadGivesUpAfterFiveTriesAndKeepsEachError(t *testing.T) {
	defer fetch.FixRetryWait(time.Millisecond)()
	m := effecttest.NewMachine()
	f := &flaky{Machine: m, fails: 9, err: errors.New("503")}
	err := fetchOnce(t.Context(), t, f, bodySum)
	if err == nil || f.calls != 5 {
		t.Errorf("err = %v after %d calls, want one after 5", err, f.calls)
	}
}

func TestDownloadDoesNotRetryAWrongSum(t *testing.T) {
	defer fetch.FixRetryWait(time.Millisecond)()
	m := effecttest.NewMachine()
	m.Pages["u"] = []byte("body")
	f := &flaky{Machine: m}
	err := fetchOnce(t.Context(), t, f, "00")
	if !errors.Is(err, effect.ErrSHA256) || f.calls != 1 {
		t.Errorf("err = %v after %d calls, want ErrSHA256 after 1", err, f.calls)
	}
}

func TestDownloadStopsOnADoneContext(t *testing.T) {
	defer fetch.FixRetryWait(time.Hour)()
	m := effecttest.NewMachine()
	f := &flaky{Machine: m, fails: 9, err: errors.New("503")}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := fetchOnce(ctx, t, f, bodySum)
	if !errors.Is(err, context.Canceled) || f.calls != 1 {
		t.Errorf("err = %v after %d calls, want canceled after 1", err, f.calls)
	}
}
