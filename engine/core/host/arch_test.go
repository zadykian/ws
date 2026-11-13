package host_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
)

func TestArchitectureIsDpkgsAndTrimmed(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("arm64\n")}
	got, err := host.Architecture(context.Background(), m)
	if err != nil || got != "arm64" {
		t.Errorf("Architecture = %q, %v, want arm64", got, err)
	}
}

func TestArchitectureNamesTheFailedRead(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Err: errors.New("no dpkg")}
	_, err := host.Architecture(context.Background(), m)
	if err == nil || !strings.Contains(err.Error(), "reading dpkg's architecture: ") {
		t.Errorf("err = %v, want the failed read named", err)
	}
}
