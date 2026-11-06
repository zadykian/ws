package wizard

import "testing"

func TestMachineOfAStepWithNone(t *testing.T) {
	if m, ok := MachineOf(nil); ok || m != "" {
		t.Errorf("MachineOf(nil) = %q, %v", m, ok)
	}
}
