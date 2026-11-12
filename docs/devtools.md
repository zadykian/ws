# devtools

The `devtools` role installs Node, the .NET SDKs, gh, glow, the libraries headless Chromium needs,
Go, gopls, Delve, roslyn-language-server, the SQL and Docker servers and rustup. It runs as Go
steps in [internal/server/devtools](../internal/server/devtools): the repositories, the packages,
Go, the servers and rustup, in this order.

Node comes from NodeSource's apt repository, glow from Charm's and gh from GitHub CLI's. Each .NET
SDK comes from Ubuntu's archive or, for those it lacks, Launchpad's dotnet/backports PPA. Each run
fetches their keys again. A key goes in only where it holds a certificate at least, and the package
pins the fingerprint of each. A subkey added to a certificate passes, and so does a key that drops a
pinned certificate. A certificate the package does not pin fails the run, in the first armored block
or in another, as apt reads them all. The package reads the key's packets itself, as `gpg` is not on
every machine. A packet it cannot parse fails the run, and only keys of version 4 and 6 pass. A
repository that is no longer needed loses its `.sources` file and its key, and so do the sources
that the repositories' own instructions add.

The packages wait on the repositories: `ws server apply --check` says "after Devtools
repositories" where a repository is still to come, as apt may not know the packages then. The
libraries for Chromium go in without recommends.

Go goes in `/usr/local/go` from go.dev's tarball, checked against the SHA-256 go.dev lists for it.
ws unpacks the tarball beside the target and moves the tree into place whole. The tree's
`.ws-source` file holds the tarball's URL and is its probe. A run cut short never passes for the
version, and a current Go makes no fetch. A new `GoVersion` in `internal/server/devtools/pins.go`
replaces the Go there, and the Go that builds gopls. rustup installs Rust's stable toolchain in the
user's home. Once there, rustup updates itself, so ws leaves it alone.

## The language servers

- gopls, Go's server, comes from `go install` at the version `pins.go` pins, with the Go of this
  role and Go's checksum database. A change of either version builds it again.
- Delve, Go's debugger, goes in the same way and a change of either version builds it again.
  Zed's debugger runs it for Go.
- roslyn-language-server, C#'s server, is the server of VS Code's C# extension. It comes from
  nuget.org with `dotnet tool` at the version `pins.go` pins, and runs on the .NET 10 of this role.
- They go in `ws_user`'s home: `~/go/bin/gopls`, `~/go/bin/dlv` and
  `~/.dotnet/tools/roslyn-language-server`. Their paths follow `ws_home`, so the `zed` and
  `fresh` roles find them with this role not run.

## The SQL and Docker servers

postgres-language-server and docker-language-server are release binaries, each pinned in
`internal/server/devtools/binaries.go` by version and by SHA-256 for `amd64` and `arm64`. The
role downloads the one for the machine, by the architecture that dpkg names, to `/usr/local/bin`,
owned by root with mode 0755, so any `ws_user` can run it. A failed download is tried five times,
ten seconds apart. A machine of another architecture fails the step. The settings name both by
path, so Zed downloads neither. Fresh names them too.
`ws server apply --only zed` does not install them: pass `--only devtools,zed`, or `devtools,fresh`.

- docker-language-server's telemetry is off, and its experimental `removeOverlappingIssues` drops
  the diagnostics that rcjsuen's server also gives. Check the option on each bump.
- SQL's formatter is postgres-language-server, as the extension's maintainer advises
  (zed-postgres-language-server#19). `format_on_save` stays `off`, Zed's default.
- `file_types` globs make `Dockerfile.dev` a Dockerfile and `compose.override.yaml` a Compose file.
  The Mac's `file_types` must stay equal, as each side detects languages for itself.
- Compose's BuildKit lint needs `docker-buildx-plugin`, which the docker role installs.

A project's `postgres-language-server.jsonc` holds `db` fields, never a `connectionString`
(zed-postgres-language-server#14):

```jsonc
{ "db": { "host": "127.0.0.1", "port": 5432, "username": "app", "database": "app" } }
```

- No secret goes in the file. With no `db.password`, the server reads `PGPASSWORD`, then
  `~/.pgpass` (mode 0600, made by hand), then falls back to `postgres`. `~/.pgpass` compares the
  host literally, so `localhost` does not match `127.0.0.1`. The file is narrower than
  `PGPASSWORD`, which reaches every libpq client.
- Code actions that run statements stay off until `db.allowStatementExecutionsAgainst` names
  schemas.
- ws drops new connections to private ranges ([outbound](security.md#outbound)): a database on
  the LAN hangs for `db.connTimeoutSecs`, 10 by default. Loopback, Docker and `ikeward0` work.
- To bump a pin, read the digests with the `gh api` command in `binaries.go` and check a download
  with `sha256sum`. A bumped postgres-language-server may need its daemon stopped.

## The npm servers

The role installs four packages into `~/.local/lib/language-servers`, owned by `ws_user`, from the
`package.json` and `package-lock.json` in `internal/server/devtools/servers`:

- `@ansible/ansible-language-server`, for Ansible.
- `@zed-industries/vscode-langservers-extracted`, whose HTML server Zed installs by default.
- `dockerfile-language-server-nodejs`, rcjsuen's server, beside Docker's own.
- `sql-formatter`, linked as `/usr/local/bin/sql-formatter`, root's, so a project's settings name
  the bare command.

`npm ci --ignore-scripts` runs as `ws_user`. It fails where the lock and `package.json` disagree or
a sha512 `integrity` differs, so these downloads are checked, though not against a SHA-256. The
step keeps `node_modules/.ws-sha256`, the digest of both files, and runs again where it differs. It
waits for the step of the Node packages, which sets up npm.

The settings name the three servers by path, with `--stdio`, so Zed skips its npm lookup of their
latest versions. They also set `node.path`, so Zed never downloads a Node. HTML formats through its
server, so Zed installs no Prettier.

To bump a package:

1. Edit its version in `package.json`.
2. In `internal/server/devtools/servers`, run `npm install --package-lock-only --ignore-scripts
   --no-audit --no-fund`.
3. Commit both files, after checking each `integrity` against
   `npm view PACKAGE@VERSION dist.integrity`.

`go test` checks that the lock pins each package at its version. ansible-language-server 26.6.0 is
npm's latest. It lacks the didSave fix (vscode-ansible#2765), so reopen a file whose lint results
vanish after an edit.

## gh

gh comes from GitHub CLI's apt repository, as the gh of Ubuntu's archive is one that GitHub's API
has left behind. Its key file holds two certificates: one that expired in 2026, and the one that
signs. The role pins both. `/etc/apt/apt.conf.d/52unattended-upgrades-github-cli` allows the
repository's origin, so unattended-upgrades installs each release of gh within a day.

A step after the packages logs gh in to github.com as the user, by `token.github` in the secrets file
([The secrets file](shell.md#the-secrets-file)). It compares `gh auth token -h github.com` with the
token. Where they differ, it runs `gh auth login -h github.com -p https --insecure-storage
--with-token`, the token on stdin. ws has no keyring, so gh writes `~/.config/gh/hosts.yml`, mode
0600. `--check` says "log gh in by token.github", never which token. gh's refusal fails the run. With
no `token.github`, or no secrets file, the step leaves gh's login alone.

gh 2.91.0 and later send telemetry unless told not to. A step before the login reads `gh config get
telemetry` as the user, and runs `gh config set telemetry disabled` where it says otherwise. Both
run with `GH_TELEMETRY=0`, so that they send none themselves. `--check` says "turn gh's telemetry
off".

## The signing key

The role does not register the key that signs the machine's commits ([Signing](shell.md#signing)).
ws's gh holds a token with repository permissions alone, which cannot add a signing key. The Mac's
wizard adds it with the Mac's gh ([Signing key](mac-wizard.md#signing-key)), or add the `.pub` by
hand in the account's settings on GitHub, as a signing key. Until then, GitHub shows the machine's
commits as unverified, and the ruleset on `main` refuses them.
