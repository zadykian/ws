package effect_test

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/effect"
)

// passLines runs the script under sh, takes the lines that start with "event ", and returns those it
// took and what reached Stderr.
func passLines(ctx context.Context, stdin, script string) ([]string, string, error) {
	var shown bytes.Buffer
	var taken []string
	take := func(line string) bool {
		if strings.HasPrefix(line, "event ") {
			taken = append(taken, line)
			return true
		}
		return false
	}
	err := effect.OSMachine{Stderr: &shown}.PassLines(ctx, []byte(stdin), take, "sh", "-c", script)
	return taken, shown.String(), err
}

func TestPassLinesKeepsWhatItTakesOffStderr(t *testing.T) {
	taken, shown, err := passLines(context.Background(), "", `echo one; echo 'event a'; echo two >&2; echo; echo 'event b'; printf last`)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"event a", "event b"}; !slices.Equal(taken, want) {
		t.Errorf("taken = %q, want %q", taken, want)
	}
	if want := "one\ntwo\n\nlast"; shown != want {
		t.Errorf("Stderr = %q, want %q", shown, want)
	}
}

func TestPassLinesTakesALineThatComesInParts(t *testing.T) {
	taken, shown, err := passLines(context.Background(), "", `printf 'eve'; sleep 0.05; printf 'nt a\nx'; sleep 0.05; printf 'y\n'`)
	if err != nil || !slices.Equal(taken, []string{"event a"}) || shown != "xy\n" {
		t.Errorf("taken = %q, Stderr = %q, err = %v", taken, shown, err)
	}
}

// A cut leaves a last line with no newline, which is offered as it stands.
func TestPassLinesOffersALastLine(t *testing.T) {
	taken, shown, err := passLines(context.Background(), "", `printf 'event cut'`)
	if err != nil || !slices.Equal(taken, []string{"event cut"}) || shown != "" {
		t.Errorf("taken = %q, Stderr = %q, err = %v", taken, shown, err)
	}
}

func TestPassLinesGivesStdinAndAnExit(t *testing.T) {
	taken, shown, err := passLines(context.Background(), "event in\n", `cat; echo out; exit 3`)
	if err == nil || !strings.Contains(err.Error(), "exit status 3") {
		t.Errorf("err = %v", err)
	}
	if !slices.Equal(taken, []string{"event in"}) || shown != "out\n" {
		t.Errorf("taken = %q, Stderr = %q", taken, shown)
	}
}

// ssh's master holds what the first ssh was given: that must not hold up a plain run.
func TestPassLinesPassesWhereAProcessHoldsThePipe(t *testing.T) {
	start := time.Now()
	_, shown, err := passLines(context.Background(), "", `sleep 8 & echo left`)
	if err != nil || shown != "left\n" {
		t.Errorf("Stderr = %q, err = %v", shown, err)
	}
	if took := time.Since(start); took > 6*time.Second {
		t.Errorf("it took %v", took)
	}
}

func TestPassLinesEndsWithItsContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, _, err := passLines(ctx, "", `exec sleep 8`)
	if err == nil {
		t.Error("a command cut by its context passed")
	}
}

type brokenOut struct{}

func (brokenOut) Write([]byte) (int, error) { return 0, errors.New("broken") }

func TestPassLinesReportsAStderrThatFails(t *testing.T) {
	left := func(string) bool { return false }
	err := effect.OSMachine{Stderr: brokenOut{}}.PassLines(context.Background(), nil, left, "sh", "-c", "echo x")
	if err == nil || !strings.Contains(err.Error(), "broken") {
		t.Errorf("err = %v", err)
	}
}
