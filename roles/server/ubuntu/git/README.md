# git

The `git` role, Go steps in `roles/server/ubuntu/git`, writes `/etc/ws/gitconfig`, the signing key,
the allowed signers, git's tokens and the clones. It runs after `shell`. `/etc/ws/gitconfig` is
root's and 644, and each run replaces any change made to it. `~/.gitconfig` stays the user's, with
one include of ws's at its top, so that the file's own settings win:

```ini
[include]
	path = /etc/ws/gitconfig
```

A run adds the include once, and makes the file where none exists. It keeps a `~/.gitconfig` that
lacks the include beside the new one, as `~/.gitconfig.*~`: the name ends in the run's time. The
dry run shows ws's lines alone, as the file may hold a credential, an `http.extraHeader` say. A
link at `~/.gitconfig` or on the way is refused. So is an include below the file's own settings,
which it then overrides, or a second one: the run stops, and the include must go to the top, once.
Where the file sets identity or signing of its own, such as `user.email` or `user.signingkey`, the
run names them: they win over `/etc/ws/gitconfig`'s. So it names those of git's XDG file,
`~/.config/git/config`, which lose to them. The Mac writes the same file at the same
path.

`/etc/ws/gitconfig` has gh's credential program already, as `gh auth setup-git` writes it, so
nothing needs running on ws. Where it runs, gh writes its two credential sections to `~/.gitconfig`
with `--global`, below the include. Their empty `helper =` line resets the helpers of
`/etc/ws/tokens.gitconfig` for the URLs under them: a token for a path on GitHub stops answering.
Remove those sections where they stand. Each run warns of a section of `~/.gitconfig` that resets
a token's helper, by the section's name alone.

`/etc/ws/gitconfig` takes its settings from `roles/internal/gitcfg`, which the Mac writes too, so
that both sides prune. It sets `fetch.prune`, so a fetch drops the remote-tracking branches that
the remote deleted, a landed pull request's say. A `false` in `~/.gitconfig` keeps them.

## Signing

`/etc/ws/gitconfig` signs each commit with an SSH key of the machine's own: the `.pub`
that `WS_GIT_SIGNING_PUB_KEY` names in the values file, `~/.ssh/id_ed25519_signing.pub` by default.
The private key lies beside it, without `.pub`. The `.pub` lies below `WS_HOME`, or the run fails.
The role makes the pair once, an ed25519 key with no passphrase, as tools and claude commit
unattended. The key only signs. The role does not register it with GitHub: [The signing key](../devtools/README.md#the-signing-key).

To revoke the key, delete it in the account's settings on GitHub, then delete
`~/.ssh/id_ed25519_signing` and its `.pub`. The next run makes a new key. The Mac's wizard
registers it, at its Signing key stage, or add the `.pub` by hand. A key deleted on GitHub alone
comes back the same way.

`git verify-commit` trusts the key for `me@zadykian.dev` through `/etc/ws/allowed_signers`,
root's and 644, the Mac's path too. A run that changes nothing reads it as the user. Each run
replaces a change made to it. A `values.yml` that names `shell_git_allowed_signers` is refused.
So `git verify-commit` fails here
on a commit another key signed, an older machine's say, though GitHub verifies it by the account's
keys.

A signing setting in `~/.gitconfig`, such as `user.signingkey`, `commit.gpgsign` or `tag.gpgsign`,
wins over the role's. The run that adds the include names any it finds, and a later run does not.
To sign with another key, set it there.

A dry run makes no key, so on a machine without one it shows the key's creation and skips the
allowed signers file.

## Git credentials

The `[git]` table of [the secrets file](../shell/README.md#the-secrets-file), empty by default, gives git a token for
each server besides GitHub. Its key is the server's URL, as `[git."https://git.example.com"]`, or a
path on it, which covers the repositories below it. Each entry takes:

- `token`: a personal access token, say;
- `username`: `token` by default, as such servers take any;
- `proactive_auth`: `true` for a server that sends no 401, so that git sends the token with its
  first request, by `http.<url>.proactiveAuth`.

A URL of `http://` fails the run, as git would send its token in clear. Each token goes in a 640
file of `/etc/ws/tokens`, a 750 directory, named for its URL. `/etc/ws/tokens.gitconfig`, which
`/etc/ws/gitconfig` includes, gets a program for each URL, which answers git's request for a
credential from that file. Root owns them all, with the account's group, so that the account reads
a token and cannot change it. The program ignores git's requests to store or erase one, so a token
the server refuses stays until the entry changes. Where two URLs match, the longer one wins. gh's
program still answers for GitHub, except below an entry's URL there.

With `--secrets`, a URL that the table lacks loses its token and its program. Without it, as in
CI's run, the role leaves the tokens and `tokens.gitconfig` as they are. No token reaches
`tokens.gitconfig`, `~/.gitconfig` or the run's output, `-v` included: a dry run shows the diff of
the credential lines, and that a token would change, not which.

To revoke a token, revoke it on the server first, then remove its entry and run again.

## Clones

`WS_GIT_CLONES` lists repositories to clone once, as the machine's `values.yml` sets it,
and is empty by default. Each entry takes:

- `repo`: the repository's URL or path, with no token, which [Git credentials](#git-credentials)
  give;
- `dest`: the clone's path, absolute or relative to `WS_HOME`.

Each repository is cloned as `WS_USER`, after the tokens are written, where its `dest` is missing.
A run never pulls, resets or removes a clone, and leaves alone a `dest` that exists, clone or not.
A private server may sit behind the VPN of the `ikeward` role, which runs before `git`
([A run](../ikeward/README.md#a-run)). A clone that fails, as from a server out of reach, fails the run, and
the next run tries again.

A dry run lists each missing `dest` without reaching the server. A URL of `http://` or `https://`
that holds a user name or token fails the run, as the clone's config would keep it. So does a
`dest` with `*`, `?` or `[`, as the check for a missing `dest` could take it as a pattern.
