package tools

import (
	"context"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

func TestWsInstallApply(t *testing.T) {
	i, m := newInstall("v1.2.3", releases, session.Status{})
	if err := i.Apply(context.Background(), step.Plan{}); err == nil || !strings.Contains(err.Error(), "install.sh failed") {
		t.Errorf("Apply where ssh has no result = %v; want the stop", err)
	}
	m.Results[sessiontest.Op("install", "v1.2.3", releases)] = effecttest.Result{}
	if err := i.Apply(context.Background(), step.Plan{}); err != nil {
		t.Errorf("Apply = %v", err)
	}
}
