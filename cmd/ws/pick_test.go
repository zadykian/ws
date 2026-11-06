package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/effect/effecttest"
)

// repeating types keys again and again, so that one lands once the list takes it.
type repeating string

func (r repeating) Read(b []byte) (int, error) {
	time.Sleep(5 * time.Millisecond)
	return copy(b, r), nil
}

func TestServerApplyOpensTheListAndLeavesWithQ(t *testing.T) {
	p, _ := serverTerminal()
	status, stdout, stderr := runWs(p, "q", "server", "apply")
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if strings.Contains(stderr, "changed:") {
		t.Errorf("a run began: %q", stderr)
	}
}

func TestServerApplyChecksFromTheListAndSaysSo(t *testing.T) {
	p, m := serverTerminal()
	m.Results["dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n tmux"] = effecttest.Result{Out: []byte("tmux ii \n")}
	var errs bytes.Buffer
	root := newRoot(p, newMacCmd(p), newServerCmd(p))
	// A scan holds c back, so each turn ends by emptying the filter that the next one types.
	status := execute(t.Context(), root, []string{"server", "apply"}, repeating("/tmux\rac/\x7f\x7f\x7f\x7f\r"), io.Discard, &errs)
	if want := "ws server apply: would change: none\n"; status != 0 || !strings.HasSuffix(errs.String(), want) {
		t.Errorf("status = %d, stderr ends %q, want %q", status, errs.String()[max(0, errs.Len()-300):], want)
	}
}
