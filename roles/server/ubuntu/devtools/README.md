# devtools

The `devtools` role installs Node, the .NET SDKs, gh, glow, the libraries headless Chromium needs,
Go, gopls, Delve, roslyn-language-server, the SQL and Docker servers and rustup. The sources, the
packages and the tools are mise's, in [mise.server.toml](mise.server.toml), and the tools' links
are units of [units.toml](units.toml). The rest runs as Go steps in
[roles/server/ubuntu/devtools](.): gh's telemetry and login, roslyn-language-server, the npm
servers and rustup.

rustup installs Rust's stable toolchain in the user's home. Once there, rustup updates itself, so
ws leaves it alone.

## The sources and the packages

`mise bootstrap` writes each source as `/etc/apt/sources.list.d/mise-NAME.sources`, with its key,
which it keeps only where its fingerprint is the one `mise.server.toml` pins. A key whose
fingerprint differs fails the run. mise never removes a source or a package. A source that a
vendor's own instructions added, such as NodeSource's setup script, leaves a file that conflicts
with ours: remove it by hand.

- Node comes from NodeSource's `node_24.x`, and the package follows that repository's latest.
- glow comes from Charm's repository, and gh from GitHub CLI's. Its key file holds two
  certificates, one that expired in 2026 and the one that signs: both are pinned.
- Each .NET SDK is a package, and Launchpad's dotnet/backports PPA gives .NET 8.
  The PPA's index for apt's amd64v3 variant lacks .NET 8. A machine that set
  `APT::Architecture-Variants` fails at this package until the setting goes.
- The libraries for Chromium are fixed packages, with apt's own recommends.

## The tools

Go, gopls, Delve, postgres-language-server and docker-language-server are tools of mise's, at the
versions that `[tools]` pins. [mise.server.lock](../../../../mise/mise.server.lock) holds the url
and the SHA-256 of each download for `linux-x64` and `linux-arm64`. Go's comes from the list that
go.dev publishes, and each server's from GitHub's digest of its release asset. A wrong one fails
the run before any unit of ws, and `MISE_LOCKED=1` refuses a tool that the lock lacks. gopls and
Delve are `go:` tools, which `go install` builds with mise's Go through Go's checksum database.
mise checks a sum at the download alone, so it mends no tool changed after its install:
`sudo env MISE_DATA_DIR=/usr/local/share/mise mise -C /opt/ws install --force TOOL` does.

- mise puts the tools under `/usr/local/share/mise/installs`, root's and mode 0755, where every
  run as root exports `MISE_DATA_DIR=/usr/local/share/mise`, as `install.sh` does. Under root's
  own data directory the account could run none. `guard/devtools-data-dir` fails a run without
  it. It runs after mise's tools, so it names the mistake, and a missing tool is then under
  `/root`: run the bootstrap again with the variable.
- Links of the file kind give each tool a path that names no version: `/usr/local/go`,
  `/usr/local/bin/gopls`, `/usr/local/bin/dlv`, `/usr/local/bin/postgres-language-server` and
  `/usr/local/bin/docker-language-server`. Each points into its version's directory, never at
  `latest`, which follows the highest version installed. shell's profile puts `/usr/local/go/bin`
  on `PATH`, and gopls runs the `go` it finds there.
- A `/usr/local/go` that is a directory fails its link: remove it by hand, with
  `rm -rf /usr/local/go`.
- To bump a tool, change its version in `[tools]` and its link in `units.toml`, then write the
  lock again in a container with git and mise:
  `MISE_ENV=server mise -C roles/server/ubuntu/devtools lock --platform linux-x64,linux-arm64`,
  then move `roles/server/ubuntu/devtools/mise.server.lock` to `mise/mise.server.lock`. `go test`
  fails where a link, the lock and `[tools]` disagree. On the server,
  `sudo env MISE_DATA_DIR=/usr/local/share/mise mise -C /opt/ws prune` drops the old versions.
- mise builds a `go:` tool once, and never again where Go changes. Delve 1.27.2 supports Go 1.25
  to 1.27, so a new minor Go goes with a newer Delve.
- roslyn-language-server, C#'s server, is the server of VS Code's C# extension. It comes from
  nuget.org with `dotnet tool` at the version `pins.go` pins, and runs on the .NET 10 of this role.
  It goes in `~/.dotnet/tools/roslyn-language-server`, `WS_USER`'s, which follows `WS_HOME`.

## The SQL and Docker servers

Zed's settings and Fresh's config name both servers by path, so Zed downloads neither.

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
- ws drops new connections to private ranges ([outbound](../security/README.md#outbound)): a database on
  the LAN hangs for `db.connTimeoutSecs`, 10 by default. Loopback, Docker and `ikeward0` work.
- A bumped postgres-language-server may need its daemon stopped.

## The npm servers

The role installs four packages into `~/.local/lib/language-servers`, owned by `WS_USER`, from the
`package.json` and `package-lock.json` in `roles/server/ubuntu/devtools/internal/npm/servers`:

- `@ansible/ansible-language-server`, for Ansible.
- `@zed-industries/vscode-langservers-extracted`, whose HTML server Zed installs by default.
- `dockerfile-language-server-nodejs`, rcjsuen's server, beside Docker's own.
- `sql-formatter`, linked as `/usr/local/bin/sql-formatter`, root's, so a project's settings name
  the bare command.

`npm ci --ignore-scripts` runs as `WS_USER`. It fails where the lock and `package.json` disagree or
a sha512 `integrity` differs, so these downloads are checked, though not against a SHA-256. The
step keeps `node_modules/.ws-sha256`, the digest of both files, and runs again where it differs. It
waits for the step of the Node packages, which sets up npm.

The settings name the three servers by path, with `--stdio`, so Zed skips its npm lookup of their
latest versions. They also set `node.path`, so Zed never downloads a Node. HTML formats through its
server, so Zed installs no Prettier.

To bump a package:

1. Edit its version in `package.json`.
2. In `roles/server/ubuntu/devtools/internal/npm/servers`, run `npm install --package-lock-only --ignore-scripts
   --no-audit --no-fund`.
3. Commit both files, after checking each `integrity` against
   `npm view PACKAGE@VERSION dist.integrity`.

`go test` checks that the lock pins each package at its version. ansible-language-server 26.6.0 is
npm's latest. It lacks the didSave fix (vscode-ansible#2765), so reopen a file whose lint results
vanish after an edit.

## gh

gh comes from GitHub CLI's apt repository, as the gh of Ubuntu's archive is one that GitHub's API
has left behind. [files/52unattended-upgrades-github-cli](files/52unattended-upgrades-github-cli)
goes to `/etc/apt/apt.conf.d/` and allows the repository's origin, so unattended-upgrades
installs each release of gh within a day.

A step after the packages logs gh in to github.com as the user, by `token.github` in the secrets file
([The secrets file](../shell/README.md#the-secrets-file)). It compares `gh auth token -h github.com` with the
token. Where they differ, it runs `gh auth login -h github.com -p https --insecure-storage
--with-token`, the token on stdin. ws has no keyring, so gh writes `~/.config/gh/hosts.yml`, mode
0600. `--check` says "log gh in by token.github", never which token. gh's refusal fails the run. With
no `token.github`, or no secrets file, the step leaves gh's login alone.

gh 2.91.0 and later send telemetry unless told not to. A step before the login reads `gh config get
telemetry` as the user, and runs `gh config set telemetry disabled` where it says otherwise. Both
run with `GH_TELEMETRY=0`, so that they send none themselves. `--check` says "turn gh's telemetry
off".

## The signing key

The role does not register the key that signs the machine's commits
([Signing](../git/README.md#signing)). ws's gh holds a token with repository permissions alone,
which cannot add a signing key. Add the `.pub` by hand in the account's settings on GitHub, as a
signing key. Until then, GitHub shows the machine's commits as unverified, and the ruleset on `main`
refuses them.
