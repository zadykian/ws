package swap

import (
	"maps"
	"testing"
)

const swapsFile = `Filename				Type		Size		Used		Priority
/swap.img                               file		4194300		1024		-2
/dev/zram0                              partition	8388604		0		100
/my\040swap                             file		10		10		-3
`

func TestSwapAreas(t *testing.T) {
	got, err := swapAreas(swapsFile)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int64{"/swap.img": 1024, "/dev/zram0": 0, "/my swap": 10}
	if !maps.Equal(got, want) {
		t.Errorf("areas = %v, want %v", got, want)
	}
	if none, err := swapAreas("Filename Type Size Used Priority\n"); err != nil || len(none) != 0 {
		t.Errorf("no areas: %v, %v", none, err)
	}
	for _, bad := range []string{"/swapfile file 1 2\n", "/swapfile file 1 x -2\n"} {
		if _, err := swapAreas(bad); err == nil {
			t.Errorf("swapAreas(%q) did not fail", bad)
		}
	}
}
