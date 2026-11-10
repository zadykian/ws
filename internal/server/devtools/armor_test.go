package devtools

import (
	"strings"
	"testing"
)

func block(headers, body, tail string) string {
	if headers != "" {
		headers += "\n"
	}
	return armorBegin + "\n" + headers + "\n" + body + "\n" + tail + armorEnd + "\n"
}

func TestDearmor(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"a body", block("", "aGVsbG8=", ""), "hello"},
		{"headers and a checksum", block("Version: GnuPG\nComment: x", "aGVsbG8=", "=AbC1\n"), "hello"},
		{"blanks around each line", "  " + block("", "aGVs\t\nbG8=\r", ""), "hello"},
		{"a header with no value", block("Charset:", "aGVsbG8=", ""), "hello"},
		{"two blocks", block("", "aGVs", "") + block("Comment: x", "bG8=", "=AbC1\n"), "hello"},
		{"blank lines between blocks", block("", "aGVs", "") + "\n \n" + block("", "bG8=", ""), "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dearmor(tt.in)
			if err != nil || string(got) != tt.want {
				t.Errorf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestDearmorRefuses(t *testing.T) {
	good := block("", "aGVsbG8=", "")
	tests := []struct{ name, in string }{
		{"nothing", ""},
		{"no first line", strings.TrimPrefix(good, armorBegin)},
		{"no last line", strings.TrimSuffix(good, armorEnd+"\n")},
		{"no blank line", armorBegin + "\naGVsbG8=\n" + armorEnd},
		{"text after the block", good + "more\n"},
		{"text between blocks", good + "more\n" + good},
		{"a block with no last line before another", strings.TrimSuffix(good, armorEnd+"\n") + good},
		{"a bad block after a good one", good + block("", "aGVs bG8=", "")},
		{"a line that is no header", block("not a header", "aGVsbG8=", "")},
		{"a line that is no base64", block("", "aGVs bG8=", "")},
		{"a base64 line with padding inside", block("", "aGVs=bG8=", "")},
		{"a body with bad padding", block("", "aGVsbG8", "")},
		{"a line of padding alone", block("", "aGVsbG8\n=", "")},
		{"a line of padding alone between blocks", block("", "aGVs\nbA\n==", "") + block("", "bG8=", "")},
		{"a private key block", strings.ReplaceAll(good, "PUBLIC", "PRIVATE")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := dearmor(tt.in); err == nil {
				t.Errorf("got %q, want an error", got)
			}
		})
	}
}
