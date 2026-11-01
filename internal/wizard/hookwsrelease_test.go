package wizard

import "testing"

func TestSumOf(t *testing.T) {
	sums := []byte("ABC123  a\n\ndef456 *./b\nnothing\n")
	for file, want := range map[string]string{"a": "abc123", "b": "def456", "c": "", "nothing": ""} {
		if got := sumOf(sums, file); got != want {
			t.Errorf("sumOf(%q) = %q; want %q", file, got, want)
		}
	}
}
