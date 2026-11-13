package setup_test

import (
	"context"
	"testing"

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
