package values

import (
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
	for _, in := range []string{"", "WS_USER: root\n", "WS_TUNNEL_PEERS: []\n", "WS_BASE_SWAP_SIZE_MB: 64\n"} {
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
