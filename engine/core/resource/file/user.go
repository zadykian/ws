package file

// Found is what a step read of one path: whether it exists and its text.
type Found struct {
	Exists bool
	Text   []byte
}
