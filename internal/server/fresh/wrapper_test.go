package fresh

import (
	"bytes"
	"context"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/step"
)

func TestTheWrapperIsWrittenForEveryUser(t *testing.T) {
	m, v := userMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results[query] = effecttest.Result{Out: []byte(version + " ii\n")}
	out := &effecttest.Out{}
	if _, err := step.Run(context.Background(), Steps(v, secrets.Secrets{}, m, out), out, false); err != nil {
		t.Fatal(err)
	}
	if got := m.Files[wrapperPath]; !bytes.Equal(got, wrapper) {
		t.Errorf("%s holds %q", wrapperPath, got)
	}
	if got := m.Modes[wrapperPath]; got != 0o755 {
		t.Errorf("mode %o", got)
	}
}
