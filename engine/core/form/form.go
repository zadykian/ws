// Package form holds what a Configure screen shows of a file's keys, whichever side's file.
package form

// Field is a key of a file as the Configure screen shows it.
type Field struct {
	Key, Text string
	// Value is what the file holds, and Default what an empty field gives.
	Value, Default string
	// Options are the values of a key that takes one of a few, and nil for a line.
	Options []string
	// Valid says why the file can't take a value. Valid is pure, as it runs on each edit.
	Valid func(string) error
}
