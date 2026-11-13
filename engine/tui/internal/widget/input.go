package widget

import (
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

// newInput returns a line of input that takes the keys at once. Its cursor stays lit, as a blink
// would wake the screen twice a second while the line waits. Bubbles' colors go, as NO_COLOR counts
// in the Theme alone, and a reversed cursor needs none.
func newInput() textinput.Model {
	input := textinput.New()
	styles := input.Styles()
	styles.Cursor.Color = nil
	plain := lipgloss.NewStyle()
	styles.Focused.Prompt, styles.Blurred.Prompt, styles.Blurred.Text = plain, plain, plain
	input.SetStyles(styles)
	_ = input.Focus()
	return input
}
