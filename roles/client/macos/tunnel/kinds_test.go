package tunnel_test

import (
	"testing"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/tunnel"
)

func TestTheRolesChangesSayTheirKind(t *testing.T) {
	tests := []struct {
		change step.Change
		want   step.ChangeKind
	}{
		{tunnel.Install{Path: "/b"}, step.Install},
		{tunnel.KeyMake{}, step.OtherChange},
	}
	for _, tt := range tests {
		if got := step.KindOf(tt.change); got != tt.want {
			t.Errorf("%T: kind = %s, want %s", tt.change, got, tt.want)
		}
	}
}
