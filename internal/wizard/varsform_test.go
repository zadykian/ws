package wizard

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// farVars returns the form of a ws whose file is file.
func farVars(file string) (VarsForm, *effecttest.Machine) {
	m := effecttest.NewMachine()
	ws := NewSession(fakeReach{Reach: MacReach{}, M: m}, Scripts{Remote: []byte("x"), Files: []byte("y")})
	m.Results[opLine("values")] = effecttest.Result{Out: []byte(file)}
	return VarsForm{Ws: ws}, m
}

// varsField returns the field of that key.
func varsField(t *testing.T, f VarsForm, key string) Field {
	fields := f.Fields()
	at := slices.IndexFunc(fields, func(field Field) bool { return field.Key == key })
	if at < 0 {
		t.Fatalf("no field %s", key)
	}
	return fields[at]
}

func TestVarsFormShowsWhatTheFileGives(t *testing.T) {
	far, _ := farVars("---\nbase_swappiness: 20\nikeward_domains: [a.example, b.example]\n")
	f, err := far.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := varsField(t, f, "base_swappiness"); got.Value != "20" || got.Text != "a whole number, default 10" ||
		got.Default != "" || got.Valid("ten") == nil {
		t.Errorf("swappiness = %+v", got)
	}
	if got := varsField(t, f, "ikeward_domains"); got.Value != "a.example, b.example" ||
		got.Text != "a list, split by commas" {
		t.Errorf("domains = %+v", got)
	}
	if got := varsField(t, f, "security_outbound"); got.Value != "true" || len(got.Options) != 2 {
		t.Errorf("outbound = %+v", got)
	}
}

func TestVarsFormLoadSaysWhyItFails(t *testing.T) {
	far, m := farVars("no_such_key: 1\n")
	if _, err := far.Load(context.Background()); err == nil || !strings.Contains(err.Error(), "no_such_key") {
		t.Errorf("a bad file: err = %v", err)
	}
	m.Results[opLine("values")] = effecttest.Result{Err: errors.New("ssh: no route\nexit status 255")}
	if _, err := far.Load(context.Background()); err == nil || !strings.Contains(err.Error(), "ssh: no route") {
		t.Errorf("no ws: err = %v", err)
	}
}

func TestVarsFormSavesTheEditsInOneLogin(t *testing.T) {
	far, m := farVars("---\nbase_swappiness: 20\n")
	f, err := far.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// ws_user=a b, and base_timezone= in front of it, as the keys go in order.
	op := []string{"values", "YmFzZV90aW1lem9uZT0=", "d3NfdXNlcj1hIGI="}
	m.Results[opLine(op...)] = effecttest.Result{}
	m.Results[opLine("values")] = effecttest.Result{Out: []byte("---\nws_user: a b\n")}
	out := &effecttest.Out{}
	f, err = f.Save(context.Background(), out, map[string]string{"ws_user": "a b", "base_timezone": ""})
	if err != nil {
		t.Fatal(err)
	}
	if got := varsField(t, f, "ws_user").Value; got != "a b" {
		t.Errorf("reads ws_user as %q", got)
	}
	if said := strings.Join(out.Lines, "\n"); !strings.Contains(said, "next ws server apply") {
		t.Errorf("said %q", said)
	}
}

func TestVarsFormSaveWithNoEditsReachesNoWs(t *testing.T) {
	far, m := farVars("---\n")
	m.Calls = nil
	out := &effecttest.Out{}
	if _, err := far.Save(context.Background(), out, map[string]string{}); err != nil || len(m.Calls) != 0 {
		t.Errorf("err = %v, calls %v", err, m.Calls)
	}
}

func TestVarsFormSaveSaysWhatWsRefused(t *testing.T) {
	far, m := farVars("---\n")
	op := []string{"values", "YmFzZV9zd2FwcGluZXNzPXRlbg=="}
	m.Results[opLine(op...)] = effecttest.Result{Err: errors.New("a whole number\nexit status 1")}
	_, err := far.Save(context.Background(), &effecttest.Out{}, map[string]string{"base_swappiness": "ten"})
	if err == nil || !strings.Contains(err.Error(), "a whole number") {
		t.Errorf("err = %v", err)
	}
}
