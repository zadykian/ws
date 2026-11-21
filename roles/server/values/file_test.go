package values

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// readAlike says whether a and b give yaml the same text. yaml writes a nil list or map as an
// empty one, as every reader of the values takes it: only decode tells them apart, for
// WS_TUNNEL_PEERS, which the file names.
func readAlike(t *testing.T, a, b Values) bool {
	textA, err := yaml.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	textB, err := yaml.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	return string(textA) == string(textB)
}

// topKeys lists the keys of a YAML mapping, in the order of its text.
func topKeys(t *testing.T, text []byte) []string {
	var doc yaml.Node
	if err := yaml.Unmarshal(text, &doc); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for i, node := range doc.Content[0].Content {
		if i%2 == 0 {
			keys = append(keys, node.Value)
		}
	}
	return keys
}

// fileOf returns the values that in gives, and their file's text.
func fileOf(t *testing.T, in string) (Values, string) {
	v, err := Parse([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	text, err := v.YAML()
	if err != nil {
		t.Fatalf("YAML for %q: %v", in, err)
	}
	return v, text
}

func TestYAMLReadsBackAsTheValues(t *testing.T) {
	for _, in := range []string{"", "WS_BASE_TIMEZONE: Asia/Tbilisi\n", "WS_TUNNEL_PEERS: []\n", "WS_BASE_SWAP_SIZE_MB: 64\n"} {
		want, text := fileOf(t, in)
		got, err := Parse([]byte(text))
		if err != nil {
			t.Fatalf("the text for %q does not read: %v\n%s", in, err, text)
		}
		if !readAlike(t, got, want) {
			t.Errorf("the text for %q reads as other values:\n%s", in, text)
		}
	}
}

// TestYAMLNamesEveryKey takes the keys from yaml's own walk of Values, so that a role's new key
// shows. A pointer is set, as yaml leaves out an unset one that omitempty marks.
func TestYAMLNamesEveryKey(t *testing.T) {
	_, text := fileOf(t, "")
	v := Default()
	size := 1
	v.Base.SwapSizeMB = &size
	whole, err := yaml.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	got, want := topKeys(t, []byte(text)), topKeys(t, whole)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("the file names %q, want %q", got, want)
	}
	if !strings.HasPrefix(text, "---\n") || !strings.Contains(text, "\nWS_BASE_SWAP_SIZE_MB: null\n") {
		t.Errorf("the file lacks its start or an unset swap size:\n%s", text)
	}
}

// TestSetFindsEachLineOfYAML holds the file in the form that values set edits: each key that a line
// holds on one line of its own, as Set writes it.
func TestSetFindsEachLineOfYAML(t *testing.T) {
	v, text := fileOf(t, "")
	for _, k := range Keys() {
		if value := v.Text(k); value != "" {
			if got, err := Set([]byte(text), k.Name, value); err != nil || string(got) != text {
				t.Errorf("set %s %q: %v; want the file unchanged:\n%s", k.Name, value, err, got)
			}
		}
		var rest []string
		for line := range strings.SplitAfterSeq(text, "\n") {
			if !strings.HasPrefix(line, k.Name+":") {
				rest = append(rest, line)
			}
		}
		if lines := strings.Count(text, "\n") - strings.Count(strings.Join(rest, ""), "\n"); lines != 1 {
			t.Errorf("%s takes %d lines, want 1", k.Name, lines)
		}
		if got, err := Set([]byte(text), k.Name, ""); err != nil || string(got) != strings.Join(rest, "") {
			t.Errorf("set %s to its default: %v; want its line out:\n%s", k.Name, err, got)
		}
	}
}
