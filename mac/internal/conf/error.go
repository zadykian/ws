package conf

import "fmt"

// Error is a refusal of mac.conf: of the line Line, or of the file as a whole where Line is 0.
type Error struct {
	Line int
	Msg  string
}

// In prints the error as the message of the file named file.
func (e Error) In(file string) string {
	if e.Line == 0 {
		return fmt.Sprintf("%s: %s", file, e.Msg)
	}
	return fmt.Sprintf("%s, line %d: %s", file, e.Line, e.Msg)
}
