package git

import (
	"bytes"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// dummy stands for a token in the fixtures: no error or output may hold it.
const dummy = "dummy-token-marker"

// tokensMachine is a machine whose TokensFile holds text, under ws's header as mise writes it.
func tokensMachine(text string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Files[TokensFile] = []byte("# Written by github.com/zadykian/ws, whose next run replaces any change made here.\n" + text)
	return m
}

const twoTokens = `["https://a.example/x"]
token = "` + dummy + `-a"
proactive_auth = true

["https://b.example"]
token = "` + dummy + `-b"
username = "bob"
`

func TestReadTokensTakesEachURL(t *testing.T) {
	tokens, present, err := ReadTokens(tokensMachine(twoTokens))
	want := map[string]Token{
		"https://a.example/x": {Token: dummy + "-a", ProactiveAuth: true},
		"https://b.example":   {Token: dummy + "-b", Username: "bob"},
	}
	if err != nil || !present || len(tokens) != len(want) {
		t.Fatalf("ReadTokens = %d tables, %v, %v; want 2, true, nil", len(tokens), present, err)
	}
	for url, tk := range want {
		if tokens[url] != tk {
			t.Errorf("%s: got another table", url)
		}
	}
}

func TestReadTokensOfAMissingFile(t *testing.T) {
	tokens, present, err := ReadTokens(effecttest.NewMachine())
	if tokens != nil || present || err != nil {
		t.Errorf("ReadTokens = %d tables, %v, %v; want none, false, nil", len(tokens), present, err)
	}
}

func TestReadTokensRefusesAnUnknownKey(t *testing.T) {
	_, _, err := ReadTokens(tokensMachine(twoTokens + "tokn = \"" + dummy + "-c\"\n"))
	if err == nil || !strings.Contains(err.Error(), TokensFile+": line 9: tokn is a key that ws does not know") ||
		strings.Contains(err.Error(), dummy) || strings.Contains(err.Error(), "b.example") {
		t.Errorf("ReadTokens = %v; want the line and the key alone", err)
	}
}

func TestReadTokensRefusesBadTOMLWithoutItsText(t *testing.T) {
	_, _, err := ReadTokens(tokensMachine(`["https://a.example"]` + "\ntoken = " + dummy + "\n"))
	if err == nil || !strings.Contains(err.Error(), "line 3") || strings.Contains(err.Error(), dummy) {
		t.Errorf("ReadTokens = %v; want the line alone", err)
	}
}

func TestReadTokensRefusesABadURL(t *testing.T) {
	for _, url := range []string{"http://a.example", "https://a b", "https://a.example?x"} {
		text := strings.ReplaceAll(twoTokens, "https://b.example", url)
		_, _, err := ReadTokens(tokensMachine(text))
		if err == nil || !strings.HasPrefix(err.Error(), TokensFile+": ") || strings.Contains(err.Error(), dummy) {
			t.Errorf("%q: ReadTokens = %v; want checkCredentials' refusal, with no token in it", url, err)
		}
	}
}

// answer runs Answer on twoTokens' file and returns what it wrote.
func answer(t *testing.T, key, action string) string {
	var out bytes.Buffer
	if err := Answer(tokensMachine(twoTokens), key, action, strings.NewReader("protocol=https\nhost=x\n\n"), &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestAnswerGivesTheKeysCredential(t *testing.T) {
	if got := answer(t, "b.example", "get"); got != "username=bob\npassword="+dummy+"-b\n" {
		t.Errorf("b.example: wrote another answer, %d bytes", len(got))
	}
	if got := answer(t, "a.example_x", "get"); got != "username=token\npassword="+dummy+"-a\n" {
		t.Errorf("a.example_x: wrote another answer, %d bytes", len(got))
	}
}

func TestAnswerIgnoresStoreAndErase(t *testing.T) {
	for _, action := range []string{"store", "erase", ""} {
		if got := answer(t, "b.example", action); got != "" {
			t.Errorf("%q: wrote %d bytes, want none", action, len(got))
		}
	}
}

func TestAnswerOfAnUnknownKeyWritesNothing(t *testing.T) {
	if got := answer(t, "c.example", "get"); got != "" {
		t.Errorf("wrote %d bytes, want none", len(got))
	}
	var out bytes.Buffer
	if err := Answer(effecttest.NewMachine(), "b.example", "get", strings.NewReader(""), &out); err != nil || out.Len() != 0 {
		t.Errorf("without the file: %v, %d bytes; want nothing", err, out.Len())
	}
}
