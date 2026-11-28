package secretstest_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/secrets"
	"github.com/zadykian/ws/macos/roles/secrets/secretstest"
)

func TestMacLandsAChangeOfTheClone(t *testing.T) {
	ctx := context.Background()
	mac := secretstest.New(effecttest.NewMachine(), map[string]string{"fnox.toml": "root = true\n"})
	mac.Keys = []secrets.DeployKey{{ID: 5, Title: "ws-a", Key: "ssh-ed25519 AAAA", ReadOnly: true}}
	repo, err := secrets.RepoOf(mac, secrets.Tree)
	if err != nil || repo.Slug() != secretstest.Slug {
		t.Fatalf("RepoOf gives %v, %v, want %s", repo, err, secretstest.Slug)
	}
	c, err := secrets.Open(ctx, mac, repo)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close(ctx)
	out, err := mac.Run(ctx, secrets.GH, "api", "--paginate", "repos/"+repo.Slug()+"/keys", "--jq",
		".[] | [.id, .title, .key, .read_only] | @tsv")
	if want := "5\tws-a\tssh-ed25519 AAAA\ttrue\n"; err != nil || string(out) != want {
		t.Errorf("gh lists the deploy keys as %q, %v, want %q", out, err, want)
	}
	if _, err := c.Fnox(ctx, nil, "set", "S1", "--provider", "age"); err != nil {
		t.Fatal(err)
	}
	ch := secrets.Change{Topic: "add-ws-a", Title: "chore: add", Body: "Added.", Files: []string{"fnox.toml"}}
	if url, err := c.Land(ctx, &strings.Builder{}, ch); url != "" || err != nil {
		t.Fatalf("Land of an unchanged clone gives %q, %v, want nothing", url, err)
	}
	mac.Machine.Files[c.Dir+"/fnox.toml"] = []byte("root = false\n")
	if url, err := c.Land(ctx, &strings.Builder{}, ch); url != secretstest.Pull || err != nil {
		t.Fatalf("Land gives %q, %v, want %s", url, err, secretstest.Pull)
	}
	landed := mac.Landed()
	if len(landed) != 1 || landed[0].Topic != ch.Topic || landed[0].Title != ch.Title ||
		!slices.Equal(landed[0].Files, ch.Files) {
		t.Errorf("the fake landed %+v, want %+v", landed, ch)
	}
	if fnoxed := mac.Fnoxed(); len(fnoxed) != 1 || !slices.Equal(fnoxed[0], []string{"set", "S1", "--provider", "age"}) {
		t.Errorf("the fake ran fnox %q, want the one set", fnoxed)
	}
	if got := mac.Files()["fnox.toml"]; got != "root = false\n" {
		t.Errorf("the clone's fnox.toml is %q", got)
	}
}

func TestMacAnswersFnoxGet(t *testing.T) {
	ctx := context.Background()
	mac := secretstest.New(effecttest.NewMachine(), map[string]string{"fnox.toml": "root = true\n"})
	c, err := secrets.Open(ctx, mac, secrets.Repo{Owner: "o", Name: "n"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close(ctx)
	if got, err := c.FnoxGet(ctx, "ws-a", "S1"); err != nil || string(got) != secretstest.Plain {
		t.Errorf("FnoxGet gives %q, %v, want Plain", got, err)
	}
	if fnoxed := mac.Fnoxed(); len(fnoxed) != 1 || !slices.Equal(fnoxed[0], []string{"-P", "ws-a", "get", "S1"}) {
		t.Errorf("the fake ran fnox %q, want the one get", fnoxed)
	}
}
