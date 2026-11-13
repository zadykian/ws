package setup_test

import (
	"testing"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/setup"
)

func TestTheRolesChangesSayTheirKind(t *testing.T) {
	tests := []struct {
		change step.Change
		want   step.ChangeKind
	}{
		{setup.Install{Path: "/b"}, step.Install},
		{setup.KeyMake{}, step.OtherChange},
	}
	for _, tt := range tests {
		if got := step.KindOf(tt.change); got != tt.want {
			t.Errorf("%T: kind = %s, want %s", tt.change, got, tt.want)
		}
	}
}
