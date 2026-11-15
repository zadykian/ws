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

func TestPortsKeepAServerApartOnlyBehindARouterItShares(t *testing.T) {
	others := []Sibling{{Name: "ws", Conf: Conf{Remote: "r.example"}}}
	// lab was placed at slot 1 while no other server shared its router, so it took the base ports.
	alone := []Sibling{{Name: "lab", Conf: Conf{Remote: "r.example", Slot: "1"}}}
	for name, tc := range map[string]struct {
		p                 Place
		remote, port, fwd string
	}{
		"alone":        {Place{Slot: 1}, "r.example", "51820", "22022"},
		"shared":       {Place{Slot: 1, Others: others}, "r.example", "51821", "22023"},
		"third":        {Place{Slot: 2, Others: others}, "r.example", "51822", "22024"},
		"other case":   {Place{Slot: 1, Others: others}, "R.Example", "51821", "22023"},
		"other router": {Place{Slot: 1, Others: others}, "s.example", "51820", "22022"},
		"no router":    {Place{Slot: 1, Others: []Sibling{{Name: "ws"}}}, "", "51820", "22022"},
		"lower slot":   {Place{Slot: 0, Others: alone}, "r.example", "51821", "22023"},
	} {
		if got := tc.p.Port(tc.remote); got != tc.port {
			t.Errorf("%s: Port = %s, want %s", name, got, tc.port)
		}
		if got := tc.p.RemotePort(tc.remote); got != tc.fwd {
			t.Errorf("%s: RemotePort = %s, want %s", name, got, tc.fwd)
		}
	}
}
