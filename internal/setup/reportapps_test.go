package setup_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

func TestACaskThatHomebrewHoldsStandsAgainstWs(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := zedStep(m, out)
	delete(m.Dirs, "/h/Applications/Zed.app")
	m.Results["brew list --cask zed"] = effecttest.Result{}
	items, err := s.Report(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if it := items[0]; it.Kind != step.PackageItem || it.Name != "zed" || it.Found != "missing" ||
		!strings.Contains(it.Conflict, "brew reinstall --cask zed") {
		t.Errorf("cask = %+v, want it missing and against ws", it)
	}
	if len(out.Lines) != 0 {
		t.Errorf("out = %q: a report says nothing", out.Lines)
	}
}

func TestACaskAbsentFromHomebrewIsPendingAlone(t *testing.T) {
	m := effecttest.NewMachine()
	s := zedStep(m, &effecttest.Out{})
	delete(m.Dirs, "/h/Applications/Zed.app")
	m.Results["brew list --cask zed"] = effecttest.Result{Err: errors.New("no cask")}
	items, err := s.Report(context.Background())
	if err != nil || items[0].Found != "missing" || items[0].Conflict != "" {
		t.Errorf("items = %+v, %v, want the cask missing, with no conflict", items, err)
	}
}
