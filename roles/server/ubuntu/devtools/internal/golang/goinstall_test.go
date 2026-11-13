package golang

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
)

func TestGoInstallSetsTheEnvironment(t *testing.T) {
	m := effecttest.NewMachine()
	user := host.User{Name: "root", Home: "/root"}
	if err := goInstall(context.Background(), m, user, "x", "/root/go/bin/x", "example.test/x@v1"); err == nil {
		t.Fatal("a program is built with no Go")
	}
	m.Files[goBin] = nil
	m.Results["env HOME=/root env GOBIN=/root/go/bin GOTOOLCHAIN=local GOENV=off GOSUMDB=sum.golang.org "+
		goBin+" install example.test/x@v1"] = effecttest.Result{}
	if err := goInstall(context.Background(), m, user, "x", "/root/go/bin/x", "example.test/x@v1"); err != nil {
		t.Fatal(err)
	}
	want := "run env HOME=/root env GOBIN=/root/go/bin GOTOOLCHAIN=local GOENV=off GOSUMDB=sum.golang.org " +
		goBin + " install example.test/x@v1"
	if !slices.Contains(m.Calls, want) {
		t.Errorf("calls %q, want %q", m.Calls, want)
	}
}
