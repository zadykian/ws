package setup_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
)

const zedConfig = "/h/.config/zed/"

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

func TestZedReportKeepsAForeignFileOut(t *testing.T) {
	m := effecttest.NewMachine()
	s := zedStep(m, &effecttest.Out{})
	m.Files[zedConfig+"keymap.json"] = []byte("[ // mine\n]\n")
	m.Files[zedConfig+"settings.json"] = []byte("{ // mine\n}\n")
	got := mactest.ReportOf(t, s)
	it := got[zedConfig+"keymap.json"]
	if !strings.Contains(it.Conflict, "not ws's") || it.Diff != "" || strings.Contains(fmt.Sprintf("%+v", it), "mine") {
		t.Errorf("keymap = %+v, want a conflict that shows none of the file", it)
	}
	if it := got[zedConfig+"settings.json"]; it.Want != "present" || it.Found != "present" || it.Conflict != "" {
		t.Errorf("settings = %+v, want it found", it)
	}
	if slices.Contains(m.Calls, "read "+zedConfig+"settings.json") {
		t.Errorf("calls = %q, which read Zed's own file", m.Calls)
	}
}

func TestZedReportShowsTheDirectoryItsPlanMakes(t *testing.T) {
	got := mactest.ReportOf(t, zedStep(effecttest.NewMachine(), &effecttest.Out{}))
	for _, name := range []string{"/h/.config", "/h/.config/zed"} {
		if it := got[name]; it.Kind != step.DirItem || it.Found != "missing" || it.Conflict != "" {
			t.Errorf("%s = %+v, want a directory, missing", name, it)
		}
	}
}
