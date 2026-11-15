package values_test

import (
	"context"
	"errors"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/form"
)

// farVars returns the form of a ws whose file is file.
func farVars(file string) (values.Form, *effecttest.Machine) {
	m := effecttest.NewMachine()
	ws := session.New(sessiontest.Reach{M: m}, session.Scripts{Remote: []byte("x"), Files: []byte("y")})
	m.Results[sessiontest.Op("values")] = effecttest.Result{Out: []byte(file)}
	return values.Form{Ws: ws}, m
}

// varsField returns the field of that key.
func varsField(t *testing.T, f values.Form, key string) form.Field {
	fields := f.Fields()
	at := slices.IndexFunc(fields, func(field form.Field) bool { return field.Key == key })
	if at < 0 {
		t.Fatalf("no field %s", key)
	}
	return fields[at]
}

func TestVarsFormShowsWhatTheFileGives(t *testing.T) {
	far, _ := farVars("---\nWS_BASE_SWAPPINESS: 20\nWS_IKEWARD_DOMAINS: [a.example, b.example]\n")
	f, err := far.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := varsField(t, f, "WS_BASE_SWAPPINESS"); got.Value != "20" || got.Text != "a whole number, default 10" ||
		got.Default != "" || got.Valid("ten") == nil {
		t.Errorf("swappiness = %+v", got)
	}
	if got := varsField(t, f, "WS_IKEWARD_DOMAINS"); got.Value != "a.example, b.example" ||
		got.Text != "a list, split by commas" {
		t.Errorf("domains = %+v", got)
	}
	if got := varsField(t, f, "WS_SECURITY_OUTBOUND"); got.Value != "true" || len(got.Options) != 2 {
		t.Errorf("outbound = %+v", got)
	}
}

func TestVarsFormLoadSaysWhyItFails(t *testing.T) {
	far, m := farVars("no_such_key: 1\n")
	if _, err := far.Load(context.Background()); err == nil || !strings.Contains(err.Error(), "no_such_key") {
		t.Errorf("a bad file: err = %v", err)
	}
	m.Results[sessiontest.Op("values")] = effecttest.Result{Err: errors.New("ssh: no route\nexit status 255")}
	if _, err := far.Load(context.Background()); err == nil || !strings.Contains(err.Error(), "ssh: no route") {
		t.Errorf("no ws: err = %v", err)
	}
}

func TestVarsFormSavesTheEditsInOneLogin(t *testing.T) {
	far, m := farVars("---\nWS_BASE_SWAPPINESS: 20\n")
	f, err := far.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// WS_USER=a b, and WS_BASE_TIMEZONE= in front of it, as the keys go in order.
	op := []string{"values", "V1NfQkFTRV9USU1FWk9ORT0=", "V1NfVVNFUj1hIGI="}
	m.Results[sessiontest.Op(op...)] = effecttest.Result{}
	m.Results[sessiontest.Op("values")] = effecttest.Result{Out: []byte("---\nWS_USER: a b\n")}
	out := &effecttest.Out{}
	f, err = f.Save(context.Background(), out, map[string]string{"WS_USER": "a b", "WS_BASE_TIMEZONE": ""})
	if err != nil {
		t.Fatal(err)
	}
	if got := varsField(t, f, "WS_USER").Value; got != "a b" {
		t.Errorf("reads WS_USER as %q", got)
	}
	if said := strings.Join(out.Lines, "\n"); !strings.Contains(said, "next ws server ubuntu apply") {
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
	op := []string{"values", "V1NfQkFTRV9TV0FQUElORVNTPXRlbg=="}
	m.Results[sessiontest.Op(op...)] = effecttest.Result{Err: errors.New("a whole number\nexit status 1")}
	_, err := far.Save(context.Background(), &effecttest.Out{}, map[string]string{"WS_BASE_SWAPPINESS": "ten"})
	if err == nil || !strings.Contains(err.Error(), "a whole number") {
		t.Errorf("err = %v", err)
	}
}
