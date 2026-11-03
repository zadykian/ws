# The Mac's secrets

The secrets of the roles are one file, `~/.config/ws/secrets.age` on the Mac: TOML that age
encrypts, in armor. Git never holds it. `ws server apply --secrets` decrypts it in its process
([The secrets file](shell.md#the-secrets-file)). A value is never in an error, a plan or a diff:
they name kinds, names, keys and paths.

## The tables

A table per kind. ws refuses a key it does not know, a key twice and a value of the wrong type. A
syntax error names its line alone.

```toml
[token]
github = "MY_TOKEN"                 # also azure_devops

[env]                               # ~/.secrets/env: any shell name
MY_TOKEN = "${token.github}"

[git."https://git.example.com"]     # an https:// URL
token = "${token.azure_devops}"
username = "bob"                    # token by default
proactive_auth = true

[file]                              # whole files, in base64
vpn_p12 = "BASE64"
vpn_sswan = "BASE64"

[password]
vpn_p12 = "PASSWORD"                # left out where the .p12 has none
```

- `[token]` names `github` and `azure_devops`, the tokens ws knows.
- `[env]` is the content of `~/.secrets/env` ([Secrets](shell.md#secrets)).
- `[git."URL"]` is a server's token in git's store ([Git credentials](shell.md#git-credentials)).
- `[file]` and `[password]` are the ikeward profile's ([The profile](ikeward.md#the-profile)). A
  TOML string can't hold the `.p12`'s bytes, so a file is base64. ws decodes it once parsed.

## References

`${kind.name}` in a string of `env`, `git` (token and username) or `password` stands for the value
of a `token`, an `env` or a `password` of that name. A reference may lead to another. `$$` is `$`,
and any other `$` fails. A file takes none.

A name that is missing, or a loop, fails the run, naming the names alone, as `env.A refers to
token.github, which is missing`.
