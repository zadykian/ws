package tui

import (
	"fmt"
	"strings"
	"time"
)

// bar draws done of total in width columns. Its two shades hold under NO_COLOR.
func bar(done, total, width int) string {
	filled := 0
	if total > 0 {
		filled = width * done / total
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// elapsed writes a time as a clock does: a first run takes 20 to 40 minutes, and a server's first
// download can pass the hour.
func elapsed(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
	}
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}
