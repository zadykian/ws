package cld_test

import (
	"context"
	"errors"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

func TestCldReportsItsVersion(t *testing.T) {
	s, m, _ := cldStep(true)
	items, err := s.Report(context.Background())
	if err != nil || items[0].Kind != step.PackageItem || items[0].Found != "installed 0.13.0" {
		t.Errorf("items = %+v, %v", items, err)
	}
	delete(m.Files, cldBin)
	delete(m.Modes, cldBin)
	if items, _ = s.Report(context.Background()); items[0].Found != "missing" {
		t.Errorf("items = %+v, want cld missing", items)
	}
}

// A report warns of nothing: a plan says the warnings, and a report only reads.
func TestCldReportWarnsNothing(t *testing.T) {
	s, m, out := cldStep(true)
	m.Results[cldBin+" --version"] = effecttest.Result{Err: errors.New(cldNo)}
	if _, err := s.Report(context.Background()); err != nil || len(out.Lines) != 0 {
		t.Errorf("Report = %v, said %q; want no warning", err, out.Lines)
	}
}
