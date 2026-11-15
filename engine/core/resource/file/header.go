package file

import "bytes"

// Mark is the words by which a run knows a file for ws's. The first line of each file that ws
// writes whole holds them, after its format's comment sign.
const Mark = "Written by github.com/zadykian/ws"

// Header returns the first line of a file that ws writes whole, sign its format's comment sign,
// such as "#" or "//". It holds no newline.
func Header(sign string) string {
	return sign + " " + Mark + ", whose next run replaces any change made here."
}

// Ours says whether ws wrote text: its first line holds Mark.
func Ours(text []byte) bool {
	line, _, _ := bytes.Cut(text, []byte("\n"))
	return bytes.Contains(line, []byte(Mark))
}
