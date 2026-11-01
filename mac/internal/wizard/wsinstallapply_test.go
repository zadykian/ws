package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

func TestWsInstallApply(t *testing.T) {
	i, m := newInstall("v1.2.3", releases, "ws -\n")
	if err := i.Apply(context.Background(), step.Plan{}); err == nil || !strings.Contains(err.Error(), "install.sh failed") {
		t.Errorf("Apply where ssh has no result = %v; want the stop", err)
	}
	m.Results[remoteLine(i.Remote, "install", "v1.2.3", releases)] = effecttest.Result{}
	if err := i.Apply(context.Background(), step.Plan{}); err != nil {
		t.Errorf("Apply = %v", err)
	}
}
