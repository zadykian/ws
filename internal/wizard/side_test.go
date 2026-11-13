package wizard

import "testing"

func TestSideOfAStepWithNone(t *testing.T) {
	if m, ok := SideOf(nil); ok || m != "" {
		t.Errorf("SideOf(nil) = %q, %v", m, ok)
	}
}
