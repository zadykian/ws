package secrets

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestVarsHoldTheNamesTheRolesRead(t *testing.T) {
	s := Secrets{Shell: Shell{SecretEnv: map[string]string{"A": "b"}}, Ikeward: Ikeward{P12Password: "pw"},
		Files: Files{P12: []byte("\x00\xffp12"), Sswan: []byte("sswan")}}
	data, err := s.Vars()
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"shell_secret_env", "ikeward_p12_password", "ikeward_p12_base64", "ikeward_sswan_base64"} {
		if _, ok := got[k]; !ok {
			t.Errorf("no %s in %s", k, data)
		}
	}
	if got["ikeward_p12_base64"] != "AP9wMTI=" || len(got) != 4 {
		t.Errorf("got %v", got)
	}
}

func TestVarsOfNothingIsEmpty(t *testing.T) {
	data, err := Secrets{}.Vars()
	if err != nil || strings.TrimSpace(string(data)) != "{}" {
		t.Errorf("got %q, %v", data, err)
	}
}
