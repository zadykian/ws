package claude

import (
	"reflect"
	"testing"
)

func TestMergeEnv(t *testing.T) {
	set := map[string]string{"B": "2", "A": "é/1"}
	cases := []struct {
		name, in, want string
		edit           envEdit
	}{
		{"empty", "", "{\n  \"env\": {\n    \"A\": \"é/1\",\n    \"B\": \"2\"\n  }\n}\n",
			envEdit{Set: []string{"A=é/1", "B=2"}}},
		{"keeps order", `{"z":1,"env":{"X":"x","B":"old"},"a":[1,2]}`,
			"{\n  \"z\": 1,\n  \"env\": {\n    \"X\": \"x\",\n    \"B\": \"2\",\n    \"A\": \"é/1\"\n  },\n  \"a\": [\n    1,\n    2\n  ]\n}\n",
			envEdit{Set: []string{"B=2", "A=é/1"}}},
		{"same", "{\"env\":{\"A\":\"\\u00e9/1\",\"B\":\"2\"}}", "{\n  \"env\": {\n    \"A\": \"é/1\",\n    \"B\": \"2\"\n  }\n}\n", envEdit{}},
		{"removes", `{"env":{"GONE":"x","A":"é/1","B":"2"}}`, "{\n  \"env\": {\n    \"A\": \"é/1\",\n    \"B\": \"2\"\n  }\n}\n",
			envEdit{Removed: []string{"GONE"}}},
		{"adds env", `{"k":true}`, "{\n  \"k\": true,\n  \"env\": {\n    \"A\": \"é/1\",\n    \"B\": \"2\"\n  }\n}\n",
			envEdit{Set: []string{"A=é/1", "B=2"}}},
	}
	for _, c := range cases {
		got, edit, err := mergeEnv([]byte(c.in), set, []string{"GONE"})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if string(got) != c.want {
			t.Errorf("%s: got\n%s\nwant\n%s", c.name, got, c.want)
		}
		if !reflect.DeepEqual(edit, c.edit) {
			t.Errorf("%s: edit %+v, want %+v", c.name, edit, c.edit)
		}
	}
}

func TestMergeEnvRefusesOtherShapes(t *testing.T) {
	for _, in := range []string{`[]`, `{"env":[]}`, `{"env":{}} x`, `{`} {
		if _, _, err := mergeEnv([]byte(in), map[string]string{"A": "1"}, nil); err == nil {
			t.Errorf("%q: no error", in)
		}
	}
}

func TestMergeEnvKeepsAKeySetAndRemoved(t *testing.T) {
	in := []byte(`{"env":{"A":"1"}}`)
	_, edit, err := mergeEnv(in, map[string]string{"A": "1"}, []string{"A"})
	if err != nil || !edit.none() {
		t.Fatalf("%+v %v", edit, err)
	}
}
