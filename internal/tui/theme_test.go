package tui

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// sgr finds the parameters of a text style's escape sequence.
var sgr = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// themeCodes returns the escape parameters that every style of theme writes.
func themeCodes(theme Theme) []string {
	var codes []string
	fields := reflect.ValueOf(theme)
	for i := range fields.NumField() {
		if style, ok := fields.Field(i).Interface().(lipgloss.Style); ok {
			for _, match := range sgr.FindAllStringSubmatch(style.Render("x"), -1) {
				codes = append(codes, strings.Split(match[1], ";")...)
			}
		}
	}
	return codes
}

// noColor is the getenv of an environment with NO_COLOR=value.
func noColor(value string) func(string) string {
	return func(string) string { return value }
}

func TestThemeWritesAnsiColorsAlone(t *testing.T) {
	colored := false
	for _, code := range themeCodes(NewTheme(noColor(""))) {
		if code == "38" || code == "48" {
			t.Errorf("code %s starts a color beyond the 16 of ANSI", code)
		}
		colored = colored || len(code) == 2
	}
	if !colored {
		t.Error("no style writes a color")
	}
}

func TestThemeHonorsNoColor(t *testing.T) {
	for _, value := range []string{"1", "yes", "0"} {
		codes := themeCodes(NewTheme(noColor(value)))
		if len(codes) == 0 {
			t.Fatalf("NO_COLOR=%s: no style writes a decoration", value)
		}
		for _, code := range codes {
			if len(code) == 2 {
				t.Errorf("NO_COLOR=%s: a style writes the color %s", value, code)
			}
		}
	}
}
