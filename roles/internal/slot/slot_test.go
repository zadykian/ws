package slot_test

import (
	"testing"

	"github.com/zadykian/ws/roles/internal/slot"
)

func TestSlotZeroKeepsTheFixedAddresses(t *testing.T) {
	var s slot.Slot
	if got := s.Net().String(); got != "10.99.0.0/24" {
		t.Errorf("Net = %s", got)
	}
	if got := s.Server().String(); got != "10.99.0.1" {
		t.Errorf("Server = %s", got)
	}
	if got := s.Mac().String(); got != "10.99.0.2" {
		t.Errorf("Mac = %s", got)
	}
}

func TestSlots(t *testing.T) {
	for _, c := range []struct {
		s             slot.Slot
		net, srv, mac string
	}{
		{1, "10.99.1.0/24", "10.99.1.1", "10.99.1.2"},
		{15, "10.99.15.0/24", "10.99.15.1", "10.99.15.2"},
	} {
		if c.s.Net().String() != c.net || c.s.Server().String() != c.srv || c.s.Mac().String() != c.mac {
			t.Errorf("slot %d = %s, %s, %s", c.s, c.s.Net(), c.s.Server(), c.s.Mac())
		}
	}
}

func TestRangeHoldsEverySlot(t *testing.T) {
	for s := slot.Slot(0); s <= slot.Last; s++ {
		if !slot.Range.Contains(s.Net().Addr()) {
			t.Errorf("slot %d is outside %s", s, slot.Range)
		}
	}
	if slot.Range.Contains(slot.Slot(slot.Last + 1).Net().Addr()) {
		t.Errorf("slot %d is inside %s", slot.Last+1, slot.Range)
	}
}

func TestParse(t *testing.T) {
	for text, want := range map[string]slot.Slot{"0": 0, "1": 1, "15": 15} {
		if got, ok := slot.Parse(text); !ok || got != want {
			t.Errorf("Parse(%q) = %d, %t", text, got, ok)
		}
	}
	for _, text := range []string{"16", "-1", "+1", "01", " 1", "1 ", "", "a", "256", "0x1"} {
		if _, ok := slot.Parse(text); ok {
			t.Errorf("Parse(%q) was taken", text)
		}
	}
}
