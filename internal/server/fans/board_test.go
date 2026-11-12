package fans

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestOnBoard(t *testing.T) {
	for _, tc := range []struct {
		name, file, vendor string
		want               bool
	}{
		{"the board", "Micro-Star International Co., Ltd.\n", "Micro-Star International Co., Ltd.", true},
		{"another board", "QEMU\n", "Micro-Star International Co., Ltd.", false},
		{"no DMI", "", "Micro-Star International Co., Ltd.", false},
		{"no vendor named", "QEMU\n", "", false},
	} {
		m := effecttest.NewMachine()
		if tc.file != "" {
			m.Files[boardVendorFile] = []byte(tc.file)
		}
		if got, err := onBoard(m, tc.vendor); err != nil || got != tc.want {
			t.Errorf("%s: onBoard = %v, %v; want %v", tc.name, got, err, tc.want)
		}
	}
}

func TestBoardNotesAnotherBoard(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Files[boardVendorFile] = []byte("QEMU\n")
	changes, err := board{m, out, "Micro-Star International Co., Ltd."}.Plan(t.Context())
	if err != nil || len(changes) != 0 || !strings.Contains(strings.Join(out.Lines, "\n"), "not one of Micro-Star") {
		t.Errorf("Plan = %v, %v; says %q", changes, err, out.Lines)
	}
}
