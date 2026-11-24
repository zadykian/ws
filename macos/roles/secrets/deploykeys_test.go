package secrets_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/secrets"
)

// gh's calls on the deploy keys of o/n.
const (
	keysLine   = secrets.GH + " api --paginate repos/o/n/keys --jq .[] | [.id, .title, .key, .read_only] | @tsv"
	postLine   = secrets.GH + " api -X POST repos/o/n/keys -f title=ws-a -f key=" + hostKey + " -F read_only=true"
	deleteLine = secrets.GH + " api -X DELETE repos/o/n/keys/7"
)

func TestDeployKeysReadsGhsLines(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[keysLine] = effecttest.Result{Out: []byte("7\tws-a\t" + hostKey + "\ttrue\n" +
		"9\tci\\tbox\t" + otherHost + " a comment\tfalse\n")}
	keys, err := secrets.DeployKeys(context.Background(), m, secrets.GH, "o/n")
	want := []secrets.DeployKey{{ID: 7, Title: "ws-a", Key: hostKey, ReadOnly: true}, {ID: 9, Title: "ci\tbox", Key: otherHost}}
	if err != nil || !slices.Equal(keys, want) {
		t.Errorf("DeployKeys gives %+v, %v, want %+v", keys, err, want)
	}
}

func TestDeployKeysOfNoneIsEmpty(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[keysLine] = effecttest.Result{}
	if keys, err := secrets.DeployKeys(context.Background(), m, secrets.GH, "o/n"); err != nil || len(keys) > 0 {
		t.Errorf("DeployKeys gives %+v, %v, want none", keys, err)
	}
}

func TestDeployKeysFails(t *testing.T) {
	for _, tc := range []struct {
		r    effecttest.Result
		want string
	}{
		{effecttest.Result{Err: errors.New("[gh api]: exit status 1: gh: Not Found (HTTP 404)")}, "admin rights"},
		{effecttest.Result{Err: errors.New("exit status 4")}, "cannot list"},
		{effecttest.Result{Out: []byte("7\tws-a\t" + hostKey + "\n")}, "3 fields"},
		{effecttest.Result{Out: []byte("x\tws-a\t" + hostKey + "\ttrue\n")}, "no number"},
		{effecttest.Result{Out: []byte("7\tws-a\t" + hostKey + "\tyes\n")}, "no boolean"},
	} {
		m := effecttest.NewMachine()
		m.Results[keysLine] = tc.r
		_, err := secrets.DeployKeys(context.Background(), m, secrets.GH, "o/n")
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("DeployKeys of %+v gives %v, want %q", tc.r, err, tc.want)
		}
	}
}

func TestPlanDeployKeys(t *testing.T) {
	inPlace := secrets.DeployKey{ID: 7, Title: "ws-a", Key: hostKey, ReadOnly: true}
	older := secrets.DeployKey{ID: 8, Title: "ws-a", Key: newerKey, ReadOnly: true}
	elsewhere := secrets.DeployKey{ID: 9, Title: "ws-old", Key: hostKey, ReadOnly: true}
	writable := secrets.DeployKey{ID: 10, Title: "ws-a", Key: hostKey}
	other := secrets.DeployKey{ID: 11, Title: "ws-b", Key: otherHost, ReadOnly: true}
	for _, tc := range []struct {
		name string
		keys []secrets.DeployKey
		drop []secrets.DeployKey
		add  bool
	}{
		{"in place", []secrets.DeployKey{other, inPlace}, nil, false},
		{"none", []secrets.DeployKey{other}, nil, true},
		{"an older key under the title", []secrets.DeployKey{older, other}, []secrets.DeployKey{older}, true},
		{"the key under another title", []secrets.DeployKey{elsewhere}, []secrets.DeployKey{elsewhere}, true},
		{"the key read-write", []secrets.DeployKey{writable}, []secrets.DeployKey{writable}, true},
		{"in place beside an older key", []secrets.DeployKey{older, inPlace}, []secrets.DeployKey{older}, false},
	} {
		drop, add := secrets.PlanDeployKeys(tc.keys, "ws-a", hostKey)
		if !slices.Equal(drop, tc.drop) || add != tc.add {
			t.Errorf("%s: PlanDeployKeys gives %+v, %v, want %+v, %v", tc.name, drop, add, tc.drop, tc.add)
		}
	}
}

func TestAddDeployKeyIsReadOnly(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[postLine] = effecttest.Result{}
	if err := secrets.AddDeployKey(context.Background(), m, secrets.GH, "o/n", "ws-a", hostKey); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(m.Calls, []string{"run " + postLine}) || !strings.HasSuffix(m.Calls[0], " -F read_only=true") {
		t.Errorf("AddDeployKey calls %q, want the read-only POST alone", m.Calls)
	}
}

func TestAddDeployKeyRefusesABadKeyOrTitle(t *testing.T) {
	m := effecttest.NewMachine()
	for _, tc := range [][2]string{{"ws-a", "ssh-rsa AAAAB3"}, {"@title", hostKey}, {"nano", hostKey}} {
		if err := secrets.AddDeployKey(context.Background(), m, secrets.GH, "o/n", tc[0], tc[1]); err == nil {
			t.Errorf("AddDeployKey of %q passes, want a refusal", tc)
		}
	}
	if len(m.Calls) > 0 {
		t.Errorf("AddDeployKey of a bad key or title ran %q", m.Calls)
	}
}

func TestDeleteDeployKey(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[deleteLine] = effecttest.Result{}
	if err := secrets.DeleteDeployKey(context.Background(), m, secrets.GH, "o/n", 7); err != nil ||
		!slices.Equal(m.Calls, []string{"run " + deleteLine}) {
		t.Errorf("DeleteDeployKey gives %v after %q, want the DELETE alone", err, m.Calls)
	}
	m.Results[deleteLine] = effecttest.Result{Err: errors.New("exit status 1")}
	if err := secrets.DeleteDeployKey(context.Background(), m, secrets.GH, "o/n", 7); err == nil {
		t.Error("DeleteDeployKey of a failed call passes, want an error")
	}
}
