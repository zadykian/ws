package configure

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/tui/internal/run"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// unread is a sheet that shows no field until it reads the file, which gives wayFile's.
func unread(reads *int, err error) Sheet {
	return Sheet{
		Name:   "far.conf",
		Fields: func(map[string]string) []widget.FieldSpec { return nil },
		Open: func(context.Context, *run.Bridge) (Sheet, error) {
			*reads++
			return wayFile(), err
		},
	}
}

func TestConfigureReadsASheetOnceItShows(t *testing.T) {
	var jobs []configureJob
	reads := 0
	s := fed(recording(&jobs, nil, wayFile(), unread(&reads, nil)))
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
	if s.saving || s.status() != "" || s.form().Name() != "HOST" {
		t.Errorf("saving = %v, status %q, field %q", s.saving, s.status(), s.form().Name())
	}
	if got := stripped(s.View().Content); !strings.Contains(got, "way.conf 1/3") {
		t.Errorf("drew:\n%s", got)
	}
	if s = fed(s, ctrlN, ctrlN); len(jobs) != 1 || s.at != 1 {
		t.Errorf("a walk back read it again: %d jobs, at %d", len(jobs), s.at)
	}
}

func TestConfigureFailedReadShowsWhyAndTriesAgain(t *testing.T) {
	var jobs []configureJob
	reads := 0
	far := unread(&reads, errors.New("ws is out of reach"))
	s := fed(recording(&jobs, nil, wayFile(), far), ctrlN)
	_, err := jobs[0].run(context.Background(), nil)
	s = fed(s, savedMsg{place: 1, err: err})
	if got := stripped(s.View().Content); !strings.Contains(got, "ws is out of reach") {
		t.Errorf("drew:\n%s", got)
	}
	if s.saving || s.status() != "not read: ctrl+s tries again" {
		t.Errorf("saving = %v, status %q", s.saving, s.status())
	}
	s = fed(s, ctrlS)
	if len(jobs) != 2 || jobs[1].sheet.Open == nil || !s.saving {
		t.Fatalf("Ctrl-S: jobs = %+v, saving = %v", jobs, s.saving)
	}
	s = fed(s, savedMsg{place: 1, err: err}, ctrlN, tea.PasteMsg{Content: "ok"})
	if s.at != 0 || s.status() == "reading" {
		t.Errorf("at %d, status %q: the first form stays usable", s.at, s.status())
	}
}

func TestConfigureAsksBeforeALossOnAnUnreadSheet(t *testing.T) {
	var jobs []configureJob
	reads := 0
	far := unread(&reads, errors.New("ws is out of reach"))
	s := fed(recording(&jobs, nil, wayFile(), far), tea.PasteMsg{Content: "x"}, ctrlN)
	s = fed(s, savedMsg{place: 1, err: errors.New("ws is out of reach")}, esc)
	if got := stripped(s.View().Content); len(jobs) != 1 || !strings.Contains(got, "esc again") {
		t.Errorf("jobs = %+v, drew:\n%s", jobs, got)
	}
}
