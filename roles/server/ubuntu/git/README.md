# git

The `git` role sets git up from `mise bootstrap`. Its `mise.server.toml` installs git and
openssh-client, and names the eight units of its `units.toml.tmpl`:

- `file/git-config`: `/etc/ws/gitconfig`, root's and 644, from `files/ws.gitconfig.tmpl`. Each
  run replaces any change made to it.
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

To revoke the key, delete it in the GitHub account's settings, then delete
`~/.ssh/id_ed25519_signing` and its `.pub`. The next run makes a new key, and its `.pub` goes to
the GitHub account by hand. A key deleted on GitHub alone needs that too.

`git verify-commit` trusts the key for `me@zadykian.dev` through `/etc/ws/allowed_signers`,
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
