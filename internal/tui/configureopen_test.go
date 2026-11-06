package tui

import (
	"context"
	"strings"
	"testing"
)

// unread is a sheet that shows no field until it reads the file, which gives wayFile's.
func unread(reads *int, err error) Sheet {
	return Sheet{
		Name:   "far.conf",
		Fields: func(map[string]string) []FieldSpec { return nil },
		Open: func(context.Context, *Bridge) (Sheet, error) {
			*reads++
			return wayFile(), err
		},
	}
}

func TestConfigureReadsASheetOnceItShows(t *testing.T) {
	var jobs []configureJob
	reads := 0
	s := fed(recording(&jobs, wayFile(), unread(&reads, nil)))
	if len(jobs) != 0 {
		t.Fatalf("a read before the sheet showed: %+v", jobs)
	}
	s = fed(s, ctrlN, press('x'))
	if len(jobs) != 1 || !s.saving || s.at != 1 || s.status() != "reading" {
		t.Fatalf("jobs = %+v, saving = %v, at %d, status %q", jobs, s.saving, s.at, s.status())
	}
	sheet, err := jobs[0].run(context.Background(), nil)
	if err != nil || reads != 1 {
		t.Fatalf("read: %v, %d reads", err, reads)
	}
	s = fed(s, savedMsg{jobs[0].place, sheet, nil})
	if s.saving || s.status() != "" || s.form().name() != "HOST" {
		t.Errorf("saving = %v, status %q, field %q", s.saving, s.status(), s.form().name())
	}
	if got := stripped(s.View().Content); !strings.Contains(got, "way.conf 1/3") {
		t.Errorf("drew:\n%s", got)
	}
	if s = fed(s, ctrlN, ctrlN); len(jobs) != 1 || s.at != 1 {
		t.Errorf("a walk back read it again: %d jobs, at %d", len(jobs), s.at)
	}
}
