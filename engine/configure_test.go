package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/tui"
)

func TestConfigureRefusesWhereTheTargetDoes(t *testing.T) {
	c, _, _ := applyCall(strings.NewReader(""))
	err := Configure(t.Context(), c, Target{Refuse: func(Call) string { return "not here" }, NoScreen: "no screen"})
	if err == nil || err.Error() != "not here" {
		t.Errorf("err = %v", err)
	}
}

func TestConfigureNeedsAScreen(t *testing.T) {
	c, _, _ := applyCall(strings.NewReader(""))
	sheets := func(context.Context, Call) ([]tui.Sheet, func(context.Context), error) { return nil, nil, nil }
	err := Configure(t.Context(), c, Target{Sheets: sheets, NoScreen: "no screen"})
	if err == nil || err.Error() != "no screen" {
		t.Errorf("err = %v", err)
	}
}
