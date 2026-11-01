package signoz

import "testing"

func TestPasswordsDiffer(t *testing.T) {
	a, errA := randomPassword()
	b, errB := randomPassword()
	if errA != nil || errB != nil || a == b {
		t.Errorf("passwords %q, %q: want two that differ (%v, %v)", a, b, errA, errB)
	}
}
