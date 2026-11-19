package secrets

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// DeployKey is a deploy key of a GitHub repository. A server fetches the secrets repository by its
// host key, which is a read-only deploy key titled with its hostname.
type DeployKey struct {
	ID       int64
	Title    string
	Key      string
	ReadOnly bool
}

// deployKeysJq turns gh's list of deploy keys into a line each, its fields split by tabs, which
// @tsv escapes inside a field.
const deployKeysJq = ".[] | [.id, .title, .key, .read_only] | @tsv"

// tsvField undoes @tsv's escapes.
var tsvField = strings.NewReplacer(`\\`, `\`, `\t`, "\t", `\n`, "\n", `\r`, "\r")

// DeployKeys lists the deploy keys of the repository slug by gh, its path gh. A key keeps its
// type and base64 alone, as a recipient does.
func DeployKeys(ctx context.Context, m effect.Machine, gh, slug string) ([]DeployKey, error) {
	out, err := m.Run(ctx, gh, "api", "--paginate", "repos/"+slug+"/keys", "--jq", deployKeysJq)
	switch {
	case err != nil && strings.Contains(err.Error(), "HTTP 404"):
		return nil, fmt.Errorf("gh's login reads no deploy keys of %s: they need its admin rights: %w", slug, err)
	case err != nil:
		return nil, fmt.Errorf("cannot list the deploy keys of %s by gh: %w", slug, err)
	}
	var keys []DeployKey
	for line := range strings.Lines(string(out)) {
		if line = strings.TrimSuffix(line, "\n"); line == "" {
			continue
		}
		k, err := deployKeyOf(line)
		if err != nil {
			return nil, fmt.Errorf("gh lists a deploy key of %s that ws cannot read: %w", slug, err)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// deployKeyOf reads a line of deployKeysJq's.
func deployKeyOf(line string) (DeployKey, error) {
	f := strings.Split(line, "\t")
	if len(f) != 4 {
		return DeployKey{}, fmt.Errorf("%d fields, not 4", len(f))
	}
	id, err := strconv.ParseInt(f[0], 10, 64)
	if err != nil {
		return DeployKey{}, fmt.Errorf("the id %q is no number", f[0])
	}
	readOnly, err := strconv.ParseBool(f[3])
	if err != nil {
		return DeployKey{}, fmt.Errorf("the key %d's read_only %q is no boolean", id, f[3])
	}
	key := strings.Fields(tsvField.Replace(f[2]))
	if len(key) > 2 {
		key = key[:2]
	}
	return DeployKey{ID: id, Title: tsvField.Replace(f[1]), Key: strings.Join(key, " "), ReadOnly: readOnly}, nil
}

// AddDeployKey adds key to the repository slug as a read-only deploy key titled title. -f passes
// each field as a string, so that none reads a file by @.
func AddDeployKey(ctx context.Context, m effect.Machine, gh, slug, title, key string) error {
	if !hostKeyForm.MatchString(key) {
		return fmt.Errorf("the deploy key %s is not ssh-ed25519 BASE64", title)
	}
	if err := CheckHostname(title); err != nil {
		return err
	}
	if _, err := m.Run(ctx, gh, "api", "-X", "POST", "repos/"+slug+"/keys", "-f", "title="+title,
		"-f", "key="+key, "-F", "read_only=true"); err != nil {
		return fmt.Errorf("cannot add the deploy key %s to %s by gh: %w", title, slug, err)
	}
	return nil
}

// DeleteDeployKey deletes the deploy key id of the repository slug.
func DeleteDeployKey(ctx context.Context, m effect.Machine, gh, slug string, id int64) error {
	if _, err := m.Run(ctx, gh, "api", "-X", "DELETE", "repos/"+slug+"/keys/"+strconv.FormatInt(id, 10)); err != nil {
		return fmt.Errorf("cannot delete the deploy key %d of %s by gh: %w", id, slug, err)
	}
	return nil
}

// PlanDeployKeys says which of keys to delete, and whether to add key under title, so that one
// read-only deploy key titled title holds key. GitHub changes no deploy key in place, and refuses
// a key that the repository holds already. So an older key under title, key under another title
// and key with write access each go before the add.
func PlanDeployKeys(keys []DeployKey, title, key string) (drop []DeployKey, add bool) {
	add = true
	for _, k := range keys {
		switch {
		case add && k.Title == title && k.Key == key && k.ReadOnly:
			add = false
		case k.Title == title || k.Key == key:
			drop = append(drop, k)
		}
	}
	return drop, add
}
