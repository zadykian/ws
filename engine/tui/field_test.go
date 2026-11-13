package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// typeIn gives the field each rune of text as a key.
func typeIn(f field, text string) field {
	for _, r := range text {
		f, _ = f.update(press(r))
	}
	return f
}

func refuseEmpty(answer string) error {
	if answer == "" {
		return errors.New("an answer is needed")
	}
	return nil
}

func TestFieldAnswerReadsAsThePlainPromptDoes(t *testing.T) {
	cases := []struct {
		name   string
		prompt prompt
		typed  string
		pasted string
		want   string
	}{
		{"typed", prompt{text: "Host"}, "ws", "", "ws"},
		{"blanks trimmed", prompt{text: "Host"}, "  ws ", "", "ws"},
		{"the default", prompt{text: "Host", fallback: "ws"}, "  ", "", "ws"},
		{"a default passed over", prompt{text: "Host", fallback: "ws"}, "mac", "", "mac"},
		{"a paste's line end", prompt{text: "Key"}, "", "SHA256:x\n", "SHA256:x"},
		{"a paste's two lines", prompt{text: "Key"}, "", "256 SHA256:x\nroot", "256 SHA256:x root"},
		{"hidden blanks kept", prompt{text: "Password", hidden: true}, " s3 ", "", " s3 "},
		{"empty and hidden", prompt{text: "Password", hidden: true}, "", "", ""},
		{"a hidden paste's line end", prompt{text: "Password", hidden: true}, "s", "3\r\n", "s3"},
		{"a hidden paste's first line", prompt{text: "Password", hidden: true}, "", "s3\nx", "s3"},
		{"a hidden paste's tab", prompt{text: "Password", hidden: true}, "", "a\tb", "a\tb"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, _ := typeIn(newField(c.prompt, window), c.typed).update(tea.PasteMsg{Content: c.pasted})
			_, got, ok := f.answer()
			if !ok || got != c.want {
				t.Errorf("answer = %q, %v; want %q", got, ok, c.want)
			}
		})
	}
}

func TestFieldErasesAHiddenAnswerAsATerminalDoes(t *testing.T) {
	f := typeIn(newField(prompt{text: "Password", hidden: true}, window), "typo")
	f, _ = f.update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	f, _ = typeIn(f, "sé").update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if _, got, _ := f.answer(); got != "s" {
		t.Errorf("past Ctrl-U and Backspace: %q", got)
	}
}

func TestFieldShowsWhatValidRefusesAndKeepsTheText(t *testing.T) {
	f := typeIn(newField(prompt{text: "Host", valid: refuseEmpty}, window), " ")
	f, _, ok := f.answer()
	if ok || !strings.Contains(f.render(testTheme, window), "an answer is needed") {
		t.Fatalf("ok = %v, and the box:\n%s", ok, f.render(testTheme, window))
	}
	f = typeIn(f, "ws")
	if _, got, ok := f.answer(); !ok || got != "ws" {
		t.Errorf("mended answer = %q, %v", got, ok)
	}
}

func TestFieldChecksTheValueAsItIsTyped(t *testing.T) {
	short := func(s string) error {
		if len(s) < 3 {
			return errors.New("too short")
		}
		return nil
	}
	shows := func(f field) bool { return strings.Contains(f.render(testTheme, window), "too short") }
	f := newField(prompt{text: "Host", valid: short}, window)
	if shows(f) {
		t.Error("a refusal showed before the first key")
	}
	height := lipgloss.Height(f.render(testTheme, window))
	if f = typeIn(f, "ab"); !shows(f) {
		t.Error("no refusal as a short value was typed")
	}
	if got := lipgloss.Height(f.render(testTheme, window)); got != height {
		t.Errorf("the box went from %d lines to %d", height, got)
	}
	if f, _ = f.update(tea.PasteMsg{Content: "c"}); shows(f) {
		t.Error("a paste left the refusal of the value it mended")
	}
}

func TestFieldKeys(t *testing.T) {
	f := newField(prompt{text: "Host"}, window)
	if _, place := f.update(enter); place != answerPlace {
		t.Errorf("Enter in the input pressed %d", place)
	}
	f, _ = f.update(tab)
	f, _ = f.update(tab)
	if got := typeIn(f, "x").input.Value(); got != "" {
		t.Errorf("a button in focus let %q into the input", got)
	}
	if _, place := f.update(enter); place != stopPlace {
		t.Errorf("Enter on the second button pressed %d", place)
	}
	f, _ = f.update(tab)
	f, _ = f.update(tea.PasteMsg{Content: "pasted"})
	if got := typeIn(f, "!").input.Value(); got != "pasted!" {
		t.Errorf("Tab back to the input, a paste and a key: %q", got)
	}
}

func TestFieldDrawsTheQuestionWithNoColorUnderNoColor(t *testing.T) {
	theme := widget.NewTheme(func(string) string { return "yes" })
	f := typeIn(newField(prompt{text: "Host", fallback: "ws"}, window), "abc")
	shown := f.render(theme, window)
	for _, want := range []string{"Host [ws]", "> ", "abc", "[enter]", "answer", "[ctrl+c]", "stop"} {
		if !strings.Contains(shown, want) {
			t.Errorf("no %q in:\n%s", want, shown)
		}
	}
	if strings.Contains(shown, "\x1b[3") {
		t.Errorf("a color under NO_COLOR: %q", shown)
	}
}

func TestFieldShowsNothingOfAHiddenAnswer(t *testing.T) {
	empty := newField(prompt{text: "Password", hidden: true}, window)
	typed := typeIn(empty, "s3cret")
	if got, want := typed.render(testTheme, window), empty.render(testTheme, window); got != want {
		t.Errorf("the box changed as a secret was typed:\n%s\nwant:\n%s", got, want)
	}
}

func TestFieldScrollsALongAnswerInsideItsBox(t *testing.T) {
	f := typeIn(newField(prompt{text: "Key"}, widget.Size{Width: 20, Height: 10}), strings.Repeat("a", 40)+"end")
	got := f.render(testTheme, widget.Size{Width: 20, Height: 10})
	if !strings.Contains(got, "end") || lipgloss.Width(got) != 20 {
		t.Errorf("the end of the answer is off the box:\n%s", got)
	}
}

func TestFieldScrollsThroughAShrink(t *testing.T) {
	small := widget.Size{Width: 30, Height: 24}
	long := typeIn(newField(prompt{text: "Key"}, window), strings.Repeat("a", 50)+"END")
	if got := long.fit(small).render(testTheme, small); !strings.Contains(got, "END") {
		t.Errorf("the end of the answer went with the shrink:\n%s", got)
	}
	inside, _ := long.update(tea.KeyPressMsg{Code: tea.KeyHome})
	for range 40 {
		inside, _ = inside.update(tea.KeyPressMsg{Code: tea.KeyRight})
	}
	shrunk := inside.fit(small)
	if got := lipgloss.Width(shrunk.input.View(testTheme, 28)); got > 28 {
		t.Errorf("past a shrink the input draws %d columns", got)
	}
	if got := shrunk.input.Input.Position(); got != 40 {
		t.Errorf("the shrink moved the cursor to %d", got)
	}
}

func TestFieldKeepsItsInputInAnyWindow(t *testing.T) {
	text := strings.Repeat("a long question ", 20)
	for w := range 14 {
		for h := range 14 {
			size := widget.Size{Width: w, Height: h}
			got := newField(prompt{text: text}, size).render(testTheme, size)
			switch {
			case w == 0 || h == 0:
				if got != "" {
					t.Errorf("%v: drew %q", size, got)
				}
			case lipgloss.Height(got) > h || lipgloss.Width(got) > w || !strings.Contains(got, ">"):
				t.Errorf("%v: no input, or past the window:\n%s", size, got)
			}
		}
	}
}

func TestShortWindowGivesUpTheButtonsThenTheFrame(t *testing.T) {
	cases := []struct {
		size    widget.Size
		want    []string
		missing []string
	}{
		{widget.Size{Width: 80, Height: 1}, []string{">"}, []string{"Host", "Question"}},
		{widget.Size{Width: 80, Height: 2}, []string{"Host", ">"}, []string{"Question"}},
		{widget.Size{Width: 80, Height: 3}, []string{"Host", ">"}, []string{"Question", "answer"}},
		{widget.Size{Width: 80, Height: 4}, []string{"Question", "Host", ">"}, []string{"answer"}},
		{widget.Size{Width: 80, Height: 5}, []string{"Question", "Host", ">", "answer"}, nil},
	}
	for _, c := range cases {
		got := newField(prompt{text: "Host"}, c.size).render(testTheme, c.size)
		for _, want := range c.want {
			if !strings.Contains(got, want) {
				t.Errorf("%v: no %q in:\n%s", c.size, want, got)
			}
		}
		for _, missing := range c.missing {
			if strings.Contains(got, missing) {
				t.Errorf("%v: %q in:\n%s", c.size, missing, got)
			}
		}
	}
}
