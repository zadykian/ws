package tui

import (
	"reflect"
	"testing"
)

func TestSpread(t *testing.T) {
	panes := []Pane{{}, {Weight: 2}}
	cases := []struct {
		total, least int
		want         []int
	}{
		{10, 2, []int{4, 6}},
		{11, 2, []int{4, 7}},
		{4, 2, []int{2, 2}},
		{3, 2, nil},
	}
	for _, c := range cases {
		if got := spread(panes, c.total, c.least); !reflect.DeepEqual(got, c.want) {
			t.Errorf("spread(%d, %d) = %v, want %v", c.total, c.least, got, c.want)
		}
	}
}
