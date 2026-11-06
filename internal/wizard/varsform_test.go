package wizard

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// farVars returns the form of a ws whose file is file.
func farVars(file string) (VarsForm, *effecttest.Machine) {
	m := effecttest.NewMachine()
	r := &Remote{SSH: SSH{M: m, Lock: "/t/l"}, Script: []byte("echo\n")}
	m.Results[remoteLine(r, "values")] = effecttest.Result{Out: []byte(file)}
	return VarsForm{Files: r}, m
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
