package launchd_test

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
)

func TestRereadSignalsEachLoadedDaemon(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/Library/LaunchDaemons"] = 0o755
	for _, n := range []string{"ws", "lab", "off"} {
		m.Files["/Library/LaunchDaemons/"+launchd.For(n).Label+".plist"] = []byte("x")
	}
	for _, n := range []string{"ws", "lab"} {
		m.Results["launchctl print system/"+launchd.For(n).Label] = effecttest.Result{}
	}
	cs, err := launchd.Reread(context.Background(), m, nil)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range cs {
		got = append(got, c.String())
	}
	want := []string{
		"would run launchctl kill SIGHUP system/" + launchd.For("lab").Label,
		"would run launchctl kill SIGHUP system/" + launchd.For("ws").Label,
	}
	if !slices.Equal(got, want) {
		t.Errorf("changes = %q, want %q", got, want)
	}
}
