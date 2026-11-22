package git

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/internal/gitcfg"
)

// logFormat has git log print NUL after each commit and US between its fields. They are the hash,
// the signature's status, the signer, the committer's email, the author's email and the values of
// the Co-authored-by trailers, RS between them. It names no key.
const logFormat = "%H%x1f%G?%x1f%GS%x1f%ce%x1f%ae%x1f%(trailers:key=Co-authored-by,valueonly,separator=%x1e)"

// shown caps the lines that a refusal lists, one for each rule a commit breaks. The rest are
// counted.
const shown = 10

// signatureStatus says what git's %G? letters mean here, where only G passes.
var signatureStatus = map[string]string{
	"N": "is missing",
	"B": "is bad",
	"E": "cannot be checked",
	"U": "is from a key that " + gitcfg.Signers + " lacks",
	"X": "is expired",
	"Y": "is from an expired key",
	"R": "is from a revoked key",
}

// isHex reports whether s is an object id's digits.
func isHex(s string) bool { return strings.Trim(s, "0123456789abcdef") == "" }

// PrePush checks the commits that a push would send. For each pushed ref, they are those that no
// remote-tracking ref holds and the remote's commit at the ref does not. refs are git's pre-push
// lines, "LOCAL-REF LOCAL-SHA REMOTE-REF REMOTE-SHA". A commit passes where its signature is good,
// its signer is its committer's email, and it holds the trailer of the maintainer's identity. The
// maintainer's own commits need no trailer: a commit of the Mac's that ws rebased, which the hook
// that adds it never saw. A deleted ref sends no commit. A tag is checked by the commit it points
// to, which an annotated tag peels to. A tag of a tree or a blob sends none. The error names each
// refused commit and its rule, never a key. ws server git pre-push runs it, from git's hook of
// ws's settings.
func PrePush(ctx context.Context, m effect.Machine, refs io.Reader) error {
	var refused []string
	seen := map[string]bool{}
	in := bufio.NewScanner(refs)
	for in.Scan() {
		f := strings.Fields(in.Text())
		if len(f) == 0 {
			continue
		}
		// A local ref may hold spaces, as the source "HEAD@{2 hours ago}" does. So the line is read
		// from its right: its last three fields are the ids and the remote ref.
		if len(f) < 4 || !isHex(f[len(f)-3]) || !isHex(f[len(f)-1]) {
			return errors.New("a line of the pushed refs is not LOCAL-REF LOCAL-SHA REMOTE-REF REMOTE-SHA")
		}
		local, remote := f[len(f)-3], f[len(f)-1]
		if strings.Trim(local, "0") == "" {
			continue
		}
		commits, err := unpushed(ctx, m, local, remote)
		if err != nil {
			return err
		}
		for _, c := range commits {
			if !seen[c.hash] {
				seen[c.hash] = true
				refused = append(refused, c.refusals()...)
			}
		}
	}
	if err := in.Err(); err != nil {
		return err
	}
	if len(refused) == 0 {
		return nil
	}
	list := refused[:min(len(refused), shown)]
	if more := len(refused) - len(list); more > 0 {
		list = append(list, fmt.Sprintf("and %d more", more))
	}
	return errors.New("the push holds commits that break a rule of ws:\n  " + strings.Join(list, "\n  ") +
		"\nSign and credit a commit on ws, or rebase one of the Mac's there (roles/server/ubuntu/git/README.md#the-push-check)")
}

// pushed is a commit that a push would send.
type pushed struct {
	hash, status, signer, committer, author string
	trailers                                []string
	unreadable                              bool
}

// unpushed lists the commits that sha leads to and neither a remote-tracking ref nor remote holds,
// the whole history where there is none. remote is the remote's object id at the ref: all zeros,
// or a commit this repository lacks, leaves out nothing. A bare clone has no tracking refs, so
// remote is all it knows of the remote. It lists none for an object that is no commit.
func unpushed(ctx context.Context, m effect.Machine, sha, remote string) ([]pushed, error) {
	// Replaced objects stay unread: a push sends the originals.
	peeled, err := m.Run(ctx, "git", "--no-replace-objects", "rev-parse", "--verify", sha+"^{}")
	if err != nil {
		return nil, err
	}
	tip := strings.TrimSpace(string(peeled))
	kind, err := m.Run(ctx, "git", "--no-replace-objects", "cat-file", "-t", tip)
	if err != nil || strings.TrimSpace(string(kind)) != "commit" {
		return nil, err
	}
	not := []string{"--not", "--remotes"}
	// A failure of cat-file means the commit is absent here, and is no error of the check.
	if strings.Trim(remote, "0") != "" {
		if _, err := m.Run(ctx, "git", "--no-replace-objects", "cat-file", "-e", remote+"^{commit}"); err == nil {
			not = append(not, remote)
		}
	}
	// --no-show-signature keeps a log.showSignature setting from printing a verdict, and a key's
	// fingerprint with it, before each record.
	args := append([]string{"--no-replace-objects", "log", "-z", "--no-show-signature",
		"--format=" + logFormat, tip}, not...)
	out, err := m.Run(ctx, "git", append(args, "--")...)
	if err != nil {
		return nil, err
	}
	var commits []pushed
	for _, rec := range strings.Split(string(out), "\x00") {
		if rec == "" {
			continue
		}
		f := strings.Split(rec, "\x1f")
		if len(f) != 6 {
			// An email or a trailer that holds US is the commit's own text: refuse the commit, as a
			// skipped record would pass it unchecked.
			commits = append(commits, pushed{hash: strings.TrimSpace(f[0]), unreadable: true})
			continue
		}
		commits = append(commits, pushed{hash: strings.TrimSpace(f[0]), status: f[1], signer: f[2],
			committer: f[3], author: f[4], trailers: strings.Split(f[5], "\x1e")})
	}
	return commits, nil
}

// refusals names each rule that the commit breaks, after its short hash.
func (c pushed) refusals() []string {
	var out []string
	name := c.hash[:min(len(c.hash), 12)]
	if c.unreadable {
		return []string{name + ": its email or a trailer holds a control character, so it cannot be checked"}
	}
	switch {
	case c.status != "G":
		why, ok := signatureStatus[c.status]
		if !ok {
			why = "is not good, status " + c.status
		}
		out = append(out, name+": signature "+why)
	case c.signer != c.committer:
		out = append(out, fmt.Sprintf("%s: signer %s is not the committer %s", name, c.signer, c.committer))
	}
	mac := gitcfg.Mac()
	if c.author != mac.Email && !slices.ContainsFunc(c.trailers, func(t string) bool {
		return strings.EqualFold(t, mac.Name+" <"+mac.Email+">")
	}) {
		out = append(out, name+": trailer Co-authored-by: "+mac.Name+" <"+mac.Email+"> is missing")
	}
	return out
}
