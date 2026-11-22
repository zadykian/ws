# git

The `git` role sets git up from `mise bootstrap`. Its `mise.server.toml` installs git, from
[the PPA that base sets up](../base/README.md#git), and openssh-client, and names the eight units of its `units.toml.tmpl`:

- `file/git-config`: `/etc/ws/gitconfig`, root's and 644, from `files/ws.gitconfig.tmpl`. Each
  run replaces any change made to it. It holds the hooks of [The co-author trailer](#the-co-author-trailer) and
  [The push check](#the-push-check).
- `guard/git-check`: `ws server git check`, after the key and its file, so that a refused
  token or clone stops no signing, and before the include, the tokens and the clones. It refuses a
  clone or a tokens file the role would act on, and names what shadows ws's settings.
- `file/git-include`: the include at the top of `~/.gitconfig`.
- `state/git-key` and `file/git-signers`: the key that signs commits and the file that trusts it
  ([Signing](#signing)).
- `state/git-tokens` and `state/git-clones`: `ws server git tokens` and `clones`, each with
  `--check` as its check ([Git credentials](#git-credentials), [Clones](#clones)).

The guard's notes, the warnings below, show in `ws server git check` and in
`ws unit read guard/git-check`. `mise bootstrap` and its plan do not show them, as mise prints
no line of a unit that passes: read the guard there after a run.

`~/.gitconfig` stays the user's, with one include of ws's at its top, so that the file's own
settings win:

```ini
[include]
	path = /etc/ws/gitconfig
```

A run adds the include once, and makes the file where none exists. It keeps a `~/.gitconfig` that
lacks the include beside the new one, as `~/.gitconfig.*~`: the name ends in the run's time. The
plan shows ws's lines alone, as the file may hold a credential, an `http.extraHeader` say. A
link at `~/.gitconfig` or on the way is refused. `ws server git included` reads the file on
its standard input and answers by its exit status. It exits 0 where the include sits at the top
once, and 1 where the file lacks it. An include below the file's own settings, which it then
overrides, or a second one, exits 2. The unit then fails and leaves the file untouched: the include
must go to the top, once.
Where the include is missing and the file sets identity or signing of its own, such as
`user.email` or `user.signingkey`, the guard names them: they win over `/etc/ws/gitconfig`'s. So
it names those of git's XDG file, `~/.config/git/config`, which lose to them. The Mac writes the
same file at the same path.

`/etc/ws/gitconfig` has gh's credential program already, as `gh auth setup-git` writes it, so
nothing needs running on ws. Where it runs, gh writes its two credential sections to `~/.gitconfig`
with `--global`, below the include. Their empty `helper =` line resets the helpers of
`/etc/ws/tokens.gitconfig` for the URLs under them: a token for a path on GitHub stops answering.
Remove those sections where they stand. The guard warns of a section of `~/.gitconfig` that
resets a token's credential helper, by the section's name alone.

`/etc/ws/gitconfig` takes its settings from `roles/internal/gitcfg`, which the Mac writes too, so
that both sides prune. It sets `fetch.prune`, so a fetch drops the remote-tracking branches that
the remote deleted, a landed pull request's say. A `false` in `~/.gitconfig` keeps them.

## Signing

`/etc/ws/gitconfig` signs each commit with an SSH key of the machine's own: the `.pub`
that `WS_GIT_SIGNING_PUB_KEY` names in the values file, `~/.ssh/id_ed25519_signing.pub` by default.
The private key lies beside it, without `.pub`. The `.pub` lies below `/root`, or the run fails,
and its path holds no `"`, `\` or control character, as the values refuse them. `state/git-key`
makes the pair once, as root, an ed25519 key with no passphrase, as tools and claude commit
unattended. It makes the key's missing directory first, 0700. The key only signs. The
role does not register it with GitHub: [The signing key](../devtools/README.md#the-signing-key).

To revoke the key, delete it in `@zadykiand`'s settings on GitHub, then delete
`~/.ssh/id_ed25519_signing` and its `.pub`. The next run makes a new key, and its `.pub` goes to
`@zadykiand` by hand. A key deleted on GitHub alone needs that too.

Each commit there is by the machine account, `zadykiand <su@zadykian.dev>`, from
`gitcfg.WS()`. `git verify-commit` trusts the key for `su@zadykian.dev` through `/etc/ws/allowed_signers`,
root's and 644, the Mac's path too. Its unit reads the `.pub` through the account, which refuses a
link on the way, and runs after the key's. Each run replaces a change made to it. A `values.yml`
that names `shell_git_allowed_signers` is refused. So `git verify-commit` fails here
on a commit another key signed, an older machine's say, though GitHub verifies it by the account's
keys.

A signing setting in `~/.gitconfig`, such as `user.signingkey`, `commit.gpgsign` or `tag.gpgsign`,
wins over the role's. The guard names any it finds while the include is missing, and not once it
is in.
To sign with another key, set it there.

A plan makes no key, so on a machine without one it shows the key's creation, and the allowed
signers file without a key. The file's write, after the key's, puts the key in.

## The co-author trailer

Each commit made on ws, by claude, lazygit, Zed or the shell, carries one trailer,
`Co-authored-by: zadykian <me@zadykian.dev>`: the maintainer, whom the machine account commits for.
The name and the email are the Mac's identity, `gitcfg.Mac()`, and a test holds the file's text
equal to it. A hook of git's config adds it, `[hook "ws-co-author"]` on `commit-msg`, which runs
`git interpret-trailers --in-place --if-exists addIfDifferent` on the message's file. `addIfDifferent`
adds the line once, so that `--amend`, a message that holds it already and a second run add none.
A message that is empty once its comments and its scissors section are cut gets none. Git runs
`commit-msg` before its check of an empty message, so a trailer alone would commit it. Git passes
the file to the command as its last argument, which the command's function takes.
claude's own trailers are off ([Attribution](../claude/README.md#attribution)).

The hook runs before a repository's own `commit-msg`, `.githooks` of ws included, and a
failing one refuses the commit. A commit that skips hooks, with `--no-verify`, has none. `git
rebase` and `git cherry-pick` run no `commit-msg`, so a commit that either writes keeps its
message, but a `reword` of `git rebase -i` runs it, as it edits the message. Hooks in git's config
came in git 2.54, which [the PPA](../base/README.md#git) gives. The test of the hook skips on an
older git, and `server`'s check of a signed commit holds the one trailer on the set-up machine
([The `server` job](../../../../.github/workflows/server.md)).

## The push check

A push from ws sends no commit that breaks a rule of ws. `[hook "ws-push"]` in `/etc/ws/gitconfig`,
on `pre-push`, runs `/usr/local/bin/ws server git pre-push`, which gets the remote and its URL as
arguments, and the pushed refs on its standard input, as git gives them. For each ref it lists
the commits that no remote-tracking ref holds, `git log LOCAL --not --remotes`, and refuses a commit
by each rule it breaks. A tracking ref of any remote holds a commit, whichever remote the push goes
to:

- `signature`: `%G?` is not `G`. A good one is one that `/etc/ws/allowed_signers` trusts.
- `signer`: the signer, `%GS`, is not the committer's email, `%ce`.
- `trailer`: it holds no `Co-authored-by: zadykian <me@zadykian.dev>`, unless its author is
  `me@zadykian.dev`. Such a commit is the Mac's, which ws rebased: the rebase signs it as ws and
  runs no `commit-msg` hook ([The co-author trailer](#the-co-author-trailer)).

A refusal names each commit by its first 12 characters and lists ten rules at most, one line for
each rule that a commit breaks, and counts the rest. It prints no key. The push stops, and the
hook's exit status is 1. These cases are decided:

- A deleted ref sends no commit, so it passes.
- A new branch is checked by the commits that no remote-tracking ref holds.
- A tag is checked by its commit, which an annotated tag peels to. The tag object's own signature
  is not checked. A tag of a tree or a blob sends no commit.
- A merge commit is checked as any commit. The commits it merges in that no remote-tracking ref
  holds are checked too.
- A commit that the Mac made and ws carries unchanged keeps the Mac's signature. ws cannot check
  it, so the check refuses it where no remote-tracking ref holds it, as after a fetch by URL. A
  fetch from a named remote gives it a tracking ref, and the check lets it pass.
  `git rebase --force-rebase` on ws signs it again.
- A commit that another remote's tracking refs hold passes in a push to this one. A commit of a
  fork, fetched and merged, goes out unchecked under a merge that is signed and credited.
- A commit whose email or a Co-authored-by trailer holds a control character, US, which the check
  reads its fields by, is refused by its name. Replaced objects (`git replace`) stay unread, as the
  push sends the originals.
- A remote with no remote-tracking ref yet has none to hold a commit, so the whole history of the
  pushed ref is checked. A fetch first gives the remote's commits their refs.
- An empty range, where every commit is held already, passes. A branch made at a pushed commit
  does.
- A line of the refs that is not four fields, or a hook that fails, such as a `ws` missing at
  its path, refuses the push.

The check stops a mistake and not a person. These skip it:

- `git push --no-verify`, and a push through GitHub's API;
- a repository's `hook.ws-push.enabled = false`, and any config that disables or replaces the hook:
  `git -c`, `GIT_CONFIG_COUNT`, or a `hook.ws-push` line in `~/.gitconfig` below the include;
- a repository's `gpg.ssh.allowedSignersFile`, which changes who `G` trusts.

GitHub's own ruleset on every branch, which the maintainer sets by hand
([How changes land](../../../../.claude/CLAUDE.md#how-changes-land)), refuses an unsigned push
whatever the machine runs. GitHub enforces the signature alone, and not the trailer.

The hook runs beside a repository's own `pre-push`, and before it.
Hooks in git's config came in git 2.54, which [the PPA](../base/README.md#git) gives. The test of
the check runs on any git with SSH signing, with keys, a config and a bare repository of its own
in a temporary directory. The test of the hook's wiring skips on an older git.

## Git credentials

git's tokens for servers besides GitHub are one secret input of the role's `mise.secrets.toml`,
`WS_GIT_TOKENS` ([Secret inputs](../frame/README.md#secret-inputs)). Its value is TOML, a table per
server's URL, or a path on it, which covers the repositories below it:

```toml
["https://git.example.com"]
token = "TOKEN"
username = "bob"        # token by default, as such servers take any
proactive_auth = true   # for a server that sends no 401
```

`proactive_auth` has git send the token with its first request, by `http.<url>.proactiveAuth`.
mise writes the value whole to `/etc/ws/git-tokens.toml`, below ws's header, and removes the file
where the input is empty. A file there by hand stops a run with the secrets task, before a
credential program takes it in. The guard refuses the file where
a key is unknown, or a token or a username is not one word. It refuses a URL that holds a space, a
quote or a backslash, and one of `http://`, as git would send its token in clear. It names a line
and a key, never a value.

`/etc/ws/tokens.gitconfig`, which `/etc/ws/gitconfig` includes, gets a credential program for each
URL of the file: `ws server git credential KEY`, where `KEY` is the URL without its scheme,
each run of other characters than letters, digits, `.`, `_` and `-` as `_`. No two URLs may give
the same key. The program names `/usr/local/bin/ws`, the machine's one ws
([ws at a fixed path](../frame/README.md#ws-at-a-fixed-path)). git runs it as root, and it reads the
URL's token from the file.
It reads no `values.yml`. Root owns both files, and the tokens' is at 0600.
The program ignores git's requests to store or erase a credential, so a token the server refuses
stays until the input changes. Where two URLs match, the longer one wins. gh's program still
answers for GitHub, except below an entry's URL there.

`tokens.gitconfig` follows the file whole: a URL the file lacks loses its program, and without the
file, `tokens.gitconfig` goes. A run without the `secrets` env, as CI's `server` bootstrap, keeps
the file, and so its programs. A plan writes no file. So in the secrets task's run, the guard and the
tokens take the input from the run's env, and the plan shows the run's change. No token reaches
`tokens.gitconfig`, `~/.gitconfig` or the run's output.

To revoke a token, revoke it on the server first, then remove its table from the input and run the
[secrets task](../secrets/README.md#the-task).

## Clones

`WS_GIT_CLONES` lists repositories to clone once, as the machine's `values.yml` sets it,
and is empty by default. Each entry takes:

- `repo`: the repository's URL or path, with no token, which [Git credentials](#git-credentials)
  give;
- `dest`: the clone's path, absolute or relative to `/root`.

Each repository is cloned as root, after the tokens are written, where its `dest` is missing.
A run never pulls, resets or removes a clone, and leaves alone a `dest` that exists, clone or not.
A private server may sit behind the VPN of the `ikeward` role
([A run](../ikeward/README.md#a-run)). A clone that fails, as from a server out of reach, fails its
unit, and the next run tries again.

`ws server git clones --check` names each missing `dest` without reaching the server. A URL
of `http://` or `https://` that holds a user name or token fails the guard, as the clone's config
would keep it. So does a
`dest` with `*`, `?` or `[`, as the check for a missing `dest` could take it as a pattern.
