package conf

import (
	"strconv"
	"testing"
)

// slots returns servers that hold the slots, one each.
func slots(held ...int) []Sibling {
	var others []Sibling
	for _, n := range held {
		others = append(others, Sibling{Name: "s" + strconv.Itoa(n), Conf: Conf{Slot: strconv.Itoa(n)}})
	}
	return others
}

func TestFreeSlotTakesTheSmallestNoServerHolds(t *testing.T) {
	for name, tc := range map[string]struct {
		others []Sibling
		want   int
	}{
		"none":         {nil, 0},
		"one at zero":  {slots(0), 1},
		"a gap":        {slots(0, 1, 3), 2},
		"zero is free": {slots(1, 2), 0},
		"no line":      {[]Sibling{{Name: "ws"}}, 1},
	} {
		got, err := FreeSlot(tc.others)
		if err != nil || int(got) != tc.want {
			t.Errorf("%s: FreeSlot = %d, %v; want %d", name, got, err, tc.want)
		}
	}
}

func TestFreeSlotRefusesWhereEverySlotIsHeld(t *testing.T) {
	all := make([]int, 16)
	for i := range all {
		all[i] = i
	}
	_, err := FreeSlot(slots(all...))
	want := "every slot, 0 to 15, holds a server: ws client macos remove NAME frees one"
	if err == nil || err.Error() != want {
		t.Errorf("FreeSlot error = %v, want %q", err, want)
	}
}
