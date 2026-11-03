package secrets

import (
	"strings"
	"testing"
)

const full = `
[token]
github = "ghp_x"
azure_devops = "az"

[env]
MY_TOKEN = "${token.github}"
plain = "a b"

[git."https://git.example.com"]
token = "${token.azure_devops}"
username = "bob"
proactive_auth = true

[file]
vpn_p12 = "AP9wMTI="
vpn_sswan = "c3N3YW4="

[password]
vpn_p12 = "pw"
`

func TestParseReadsTheKinds(t *testing.T) {
	s, err := Parse([]byte(full))
	if err != nil {
		t.Fatal(err)
	}
	g := s.Git["https://git.example.com"]
	if s.Token.GitHub != "ghp_x" || s.Env["MY_TOKEN"] != "ghp_x" || s.Env["plain"] != "a b" ||
		g.Token != "az" || g.Username != "bob" || !g.ProactiveAuth ||
		string(s.File.VPNP12) != "\x00\xffp12" || string(s.File.VPNSswan) != "sswan" || s.Password.VPNP12 != "pw" {
		t.Errorf("got %+v", s)
	}
}

func TestParseRefuses(t *testing.T) {
	//nolint:gosec // G101: inputs of a test, each a made-up value that the parser must refuse
	for name, in := range map[string]string{
		"unknown table":  "[tokens]\ngithub = \"x\"\n",
		"unknown token":  "[token]\ngitlab = \"x\"\n",
		"unknown field":  "[git.\"https://a.example\"]\nstate = \"absent\"\n",
		"key twice":      "[env]\nA = \"1\"\nA = \"2\"\n",
		"wrong type":     "[env]\nA = 1\n",
		"env name":       "[env]\n\"a-b\" = \"1\"\n",
		"env digit":      "[env]\n\"1a\" = \"1\"\n",
		"http url":       "[git.\"http://a.example\"]\ntoken = \"t\"\n",
		"url no scheme":  "[git.\"a.example\"]\ntoken = \"t\"\n",
		"base64":         "[file]\nvpn_p12 = \"not base64!\"\n",
		"stray dollar":   "[env]\nA = \"a$b\"\n",
		"file reference": "[token]\ngithub = \"x\"\n[env]\nA = \"${file.vpn_p12}\"\n",
		"missing":        "[env]\nA = \"${token.github}\"\n",
		"missing env":    "[env]\nA = \"${env.B}\"\n",
	} {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestParseEmptyGivesNothing(t *testing.T) {
	s, err := Parse(nil)
	if err != nil || s.Env != nil || s.Git != nil || s.Token.GitHub != "" {
		t.Errorf("got %+v, %v", s, err)
	}
}

func TestParseErrorsShowNoValue(t *testing.T) {
	for name, in := range map[string]string{
		"syntax":  "[env]\nA = hunter2\n",
		"unknown": "[env]\nA = \"b\"\n[tokn]\nhunter2 = \"hunter3\"\n",
		"type":    "[env]\nA = 1234\nB = [\"hunter2\"]\n",
		"dollar":  "[env]\nA = \"hunter2$\"\n",
		"base64":  "[file]\nvpn_p12 = \"hunter2!\"\n",
		"url":     "[git.\"hunter2\"]\ntoken = \"hunter3\"\n",
		"name":    "[env]\n\"hunter2-\" = \"hunter3\"\n",
		"ref":     "[env]\nA = \"hunter2${token.}\"\n",
	} {
		_, err := Parse([]byte(in))
		if err == nil {
			t.Errorf("%s: no error", name)
			continue
		}
		if strings.Contains(err.Error(), "hunter3") || strings.Contains(err.Error(), "1234") ||
			(name != "url" && name != "name" && strings.Contains(err.Error(), "hunter2")) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestParseErrorNamesTheLine(t *testing.T) {
	_, err := Parse([]byte("[env]\nA = \"b\"\nB = nope\n"))
	if err == nil || !strings.HasPrefix(err.Error(), "line 3: ") {
		t.Errorf("got %v", err)
	}
}

func TestMarshalKeepsReferencesAndEncodesFiles(t *testing.T) {
	d, err := Decode([]byte(full))
	if err != nil {
		t.Fatal(err)
	}
	out, err := Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`MY_TOKEN = '${token.github}'`, `vpn_p12 = 'AP9wMTI='`, `[git.'https://git.example.com']`} {
		if !strings.Contains(string(out), want) && !strings.Contains(strings.ReplaceAll(string(out), `"`, `'`), want) {
			t.Errorf("no %s in\n%s", want, out)
		}
	}
	back, err := Parse(out)
	want, _ := Parse([]byte(full))
	if err != nil || back.Env["MY_TOKEN"] != "ghp_x" || string(back.File.VPNP12) != string(want.File.VPNP12) {
		t.Errorf("round trip: %+v, %v", back, err)
	}
}

func TestMarshalEmptyParsesAsEmpty(t *testing.T) {
	out, err := Marshal(Secrets{})
	if err != nil {
		t.Fatal(err)
	}
	if s, err := Parse(out); err != nil || s.Env != nil || s.File.VPNP12 != nil {
		t.Errorf("got %q, %+v, %v", out, s, err)
	}
}
