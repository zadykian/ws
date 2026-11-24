# CLAUDE.md

ws sets up the maintainer's development server from code: `mise bootstrap`, which `ws ubuntu guest setup`
runs on the machine itself, each role as mise's config and ws's units. A host, which runs the guests,
has `ws ubuntu host setup` and its own roles. Its roles set up packages,
SSH by key, the links to the Mac, an IKEv2 VPN where a profile is given, and the
fans. Then the shell, git,
tmux, lazygit, Fresh, Docker, claude, SigNoz with its collector, and Zed's server. `macos/`
sets up the Mac that works on the server. The rest is the docs, the lint gates' tools and CI.

## Layout

- `ubuntu/roles/NAME`: a role per directory, with its page, `README.md`, and a line in the
  root README. It holds its mise files by env, `mise.guest.toml` for the guest,
  `mise.host.toml` for the host and `mise.self-contained.toml` for both where they run SigNoz, and
  its spec of units where it has any. The
  units run in the order of their `after`
  ([The order](../ubuntu/roles/frame/README.md#the-order)). Each role
  holds its pins, and its templates and files beside its spec. ws-fans, the fans' service, is
  `ws ubuntu host fans`, in `ubuntu/roles/fans/daemon`.
- `ubuntu/`: Ubuntu's side, its roles in `ubuntu/roles/`. Beside them sit the packages that
  serve them all, `cli` and `values`, the runbook, and `autoinstall.yaml`, the installer's
  answers ([The install job](../.github/workflows/release.md#the-install-job)). `guest/` is the
  guest's run directory, `mise -C /opt/ws/ubuntu/guest`, with its `conf.d` link, its lock and
  `mise-tasks/secrets`, the task that runs the bootstrap under ws's secrets
  ([The task](../ubuntu/roles/secrets/README.md#the-task)). `host/` is the host's, `mise -C
  /opt/ws/ubuntu/host`, with its `conf.d` link. A role's mise file names its side. `values` types
  each side's values: `/etc/ws/values.yml`, on the machine alone, such as the cable's port on a host
  ([The cable](../ubuntu/roles/cable/README.md#the-cable)).
- `go.mod` and `go.sum`: the Go module `github.com/zadykian/ws`, at the root.
  `cmd/ws/main.go` runs `cmd`, which composes the root and the exit codes with the two sides'
  commands. `engine/cli` builds the root and the exit codes. `engine/core` holds `step`,
  `effect`, with `effect/effecttest`, `effect/local` and `effect/units`, and `resource`, with
  `resource/resourcetest`, the resources the kinds share, `kind`, the kinds that a role's
  `units.toml.tmpl` names as units ([kind](../engine/core/kind/README.md)), and `update`, the
  download, checksum and replace of `ws update`. They are public, as the roles
  import them from outside `engine/`.
- `mise.toml`: mise's config that both sides share, its settings. ws is no tool of mise's: the
  machine's one ws is `/usr/local/bin/ws`. `mise.guest.toml` is
  the guest's part and `mise.host.toml` the host's, and `plugins/ws` the plugin that hands ws's
  units to `ws unit`. A role holds
  its spec, `units.toml.tmpl`, where it has units, and its mise files by env beside its page. Each
  side's `mise/`, `ubuntu/guest/mise/` and `macos/mise/`, holds one link, `conf.d`, to its
  directory of roles, and its lock
  ([The frame](../ubuntu/roles/frame/README.md)). The roles' tools lock in
  `ubuntu/guest/mise/mise.guest.lock`. `mise.macos.toml` is the Mac's part, and
  `macos/roles/frame` and `macos/facts` its frame and its units' values
  ([The Mac's frame](../macos/roles/frame/README.md)). `macos/mise/mise.macos.lock`
  locks the Mac's tools, gh among them, by their SHA-256.
- `GLOSSARY.md`: the words of Ubuntu's side and of the Mac's, such as the account and the
  home file.
- `install.sh`: installs ws's binary alone, as `/usr/local/bin/ws`, on Linux and the Mac
  ([Releases](../.github/workflows/release.md)).
- `macos/`: the Mac's side ([The Mac](../macos/README.md)), its roles in `macos/roles/`.
  Above them sit the packages that serve them all, `cli`, `values`, `facts` and `internal`, and
  the runbook. Its roles' units write `ghostty/files/ws.ghostty`, Ghostty's settings,
  `zed/files/`, Zed's settings and keymap, and `zsh/files/zshrc`, the zsh hook that turns off the
  modes that a drop leaves on.
- `internal/`: what both sides share, as `setup` and `gitcfg`. It names neither side.
- The pages sit beside their code, as the `README.md` of a role or a package. A second page stands
  beside one where the cap keeps two apart, as [docs.md](rules/docs.md) lists them.
  `.tools/README.md` holds the gates and `.github/workflows/README.md` CI.
- `.tools/`: the gates' tools. `run` pins the binaries, `files` lists a gate's files and `sizecheck`
  caps them. `build` builds ws for a target, `install-build` installs the tree's build as `/usr/local/bin/ws`,
  for CI and a trial, and `vale-comments` runs Vale on files' comments. Go's
  `./...` leaves out a dot directory, so the Go gates name the tools' packages too. `lint/` holds
  the linters' configs, which the Makefile passes by flag, but tombi's, which tombi finds as it runs
  there.
- `.githooks/`: the hooks that check each commit for the words of the local list and for secrets.
- `.claude/`: this file, the rules of `rules/`, Claude Code's `settings.json` and the lint
  plugin's `lint-gates` ([Where else to look](#where-else-to-look)).
- `.github/workflows/`: `ci.yml`, with the jobs `lint`, `server`, `host`, `install` and
  `signoz`;
  `client.yml`, with `client`, whose cases are `.github/client/NN-*.sh`; `client-cld.yml`, with
  `client-cld`; `rules.yml`,
  the checks of zadykian/toolkit on a pull request; and `issues.yml`, its check of an issue. The
  toolkit's checks warn alone. `words.yml`, with `words`, checks a pull request and an issue for the
  list's words ([The `words` job](../.github/workflows/README.md#the-words-job)).
  `release.yml`, on a tag `v*`, has `check`, `ci`, `client`, `build` and `publish`.
- `.github/pull_request_template.md` and `.github/ISSUE_TEMPLATE/`: the toolkit's templates, which
  its `tools/pin` writes, never edited here ([The rules](../.github/workflows/README.md#the-rules)).

## Commands

```sh
make lint                         # every gate, as CI's lint job runs it; -k goes past a failure
make vale FILES=README.md         # one gate alone; the gates that check files take FILES
make shell                        # ShellCheck and shfmt -d -i 4 on every shell script
make gotest                       # the Go gates run at the root, the module; so do govet and gofmt
```

The gates are `shell`, `yamllint`, `tombi`, `stylua`, `sizecheck`, `vale`, `lychee`, `rumdl`,
`actionlint`, `zizmor`, `gitleaks`, `gofmt`, `govet`, `golangci-lint`, `govulncheck` and `gotest`
([The gates](../.tools/README.md)). `make lint` needs Python 3.12 to 3.14 and unzip, on Linux or
on macOS on arm64.
It installs yamllint into `.cache`, and `.tools/run` downloads the rest there.

The tests are CI's jobs ([Checks](../.github/workflows/README.md)). `server` builds ws and runs
`ws ubuntu guest setup` twice as root on a booted runner, with setup's plan. It checks the config that
Fresh reads too, Docker's forward policy and published ports, and the secrets task
([The server job](../.github/workflows/server.md)). `host`, a try that fails no pull request, runs
`ws ubuntu host setup` twice the same way, with fans off by their fact
([The host job](../.github/workflows/README.md#the-host-job)).
`signoz` forges SigNoz's compose files again, and `install` tries `install.sh`, `ws update` and
the autoinstall file, as the `client` case 16-install does for `install.sh` on macOS. `client`
sets up a Mac on GitHub's macOS runner by `ws macos setup`, whose last run must call no sudo and whose
`--plan` must exit 0. `client-cld` sets up
Ghostty, Zed and cld by setup on another ([Ghostty and cld on the Mac](../.github/workflows/client-cld.md)). A run of `mise
bootstrap` sets up the machine that runs it, as below.

The hooks check a commit for the local list's words. This checks any other text, such as a pull
request's body, as they do, and prints 0 where it holds none:

```sh
LC_ALL=C tr _ ' ' <FILE | grep -ciwF -f <(sed -e 'y/_/ /' -e 's/^[[:space:]]*//' \
    -e 's/[[:space:]]*$//' -e '/^$/d' -e '/^#/d' ~/.config/ws/forbidden-words)
```

## How changes land

- Pull requests are stacked, each on the one below it. CI runs `lint` on each, whatever its
  base, `install` and `signoz` on each of a stack on `main`, and the longest jobs on that stack's
  top one alone
  ([Checks](../.github/workflows/README.md#checks)).
- They land on `main` by GitHub's merge commits, as
  [`/flow:land`](https://github.com/zadykian/toolkit/blob/main/skills/land/SKILL.md) does. A pull
  request's signed commits land as they are, under a merge commit that GitHub signs.
- The ruleset on `main` allows the merge method alone. It requires signed commits, the `lint`
  check, and branches up to date, so that a merge's tree is the tree CI checked.
- A pull request that changes `.github/workflows` lands like any other.
- `.claude/settings.json` denies the usual forms of a push to `main`, but not `gh pr merge`, which
  `/flow:land` runs. A rule matches the command as written, so the ruleset on `main` stays the
  guard.

## Hard rules

Each rule links where the repository gives its reason: a doc, a comment or a CI job.

- `mise bootstrap` sets up the machine that runs it. CI's `server` job runs it on a booted runner
  ([The server job](../.github/workflows/server.md)).
- A second run changes nothing: CI's `server` job runs `ws ubuntu guest setup` twice. It fails where
  `ws ubuntu guest setup --plan` does not exit 0, after the first run and after the second
  ([server-checks.sh](../.github/server-checks.sh)). CI's `client` job asks the same of
  `ws macos setup --plan` after its first run, and no sudo of its second
  ([28-bootstrap.sh](../.github/client/28-bootstrap.sh)).
- `release.yml` checks a release's files and notes for the local list's words before it publishes
  them. The list is the secret `FORBIDDEN_WORDS`, and the job fails on an empty one
  ([Cutting a release](../.github/workflows/release.md#cutting-a-release)).
- A pinned download is checked against its SHA-256 before use: ikeward's
  binary ([units.toml.tmpl](../ubuntu/roles/ikeward/units.toml.tmpl)) and otelcol's `.deb`
  ([units.toml.tmpl](../ubuntu/roles/otelcol/units.toml.tmpl)). So is Fresh's `.deb`
  ([units.toml.tmpl](../ubuntu/roles/fresh/units.toml.tmpl)), and so are .NET's SDKs, on
  the server and the Mac ([devtools](../ubuntu/roles/devtools/units.toml.tmpl),
  [dotnet](../macos/roles/dotnet/units.toml.tmpl)). So is nct6687's source
  ([fans](../ubuntu/roles/fans/units.toml.tmpl)), and so are foundryctl
  ([ci.yml](../.github/workflows/ci.yml)) and the gates' tools ([.tools/run](../.tools/run)). So is ws's
  binary: `install.sh` and `ws update` check it against the release's `ws.sha256`
  ([release.yml](../.github/workflows/release.yml), [update.go](../engine/core/update/update.go)),
  on the Mac too. So is the release's `install.sh`, which the autoinstall file checks against the
  SHA-256 that `.github/autoinstall.sh` fills in
  ([autoinstall.yaml](../ubuntu/autoinstall.yaml)). So is mise's binary. `ws ubuntu guest setup`
  and `ws macos setup` hold its SHA-256 for each target ([mise.go](../internal/setup/mise.go)).
  So is the server's fnox:
  GitHub's digest of its release asset for each platform is in `ubuntu/guest/mise/mise.guest.lock`
  ([secrets](../ubuntu/roles/secrets/README.md#fnox)). So is
  yamllint ([lint-requirements.in](../.github/lint-requirements.in)), and so are the Mac's gh, fnox
  and age-plugin-yubikey, by `macos/mise/mise.macos.lock` ([git](../macos/roles/git/README.md#gh),
  [secrets](../macos/roles/secrets/README.md#the-tools)). The lock's SHA-256 of
  age-plugin-yubikey is GitHub's digest, as its publisher lists none. So is Secretive's
  `Secretive.zip`, by the SHA-256 that mise's `macos-app` pin holds
  ([mise.macos.toml](../macos/roles/ssh/mise.macos.toml)), and its notarization, by `spctl` where `ws macos setup` installs it
  ([setupkey.go](../macos/cli/setupkey.go)). The
  Mac's casks, Ghostty, Zed and their font, take their latest by mise's `brew-cask`, outside the rule
  ([ghostty](../macos/roles/ghostty/README.md#ghostty)). The Mac's formulae, devtools' and docker's,
  take their latest by mise's `brew`, each checked by the SHA-256 of Homebrew's own bottle
  ([devtools](../macos/roles/devtools/README.md#the-formulae),
  [docker](../macos/roles/docker/README.md#the-formulae)). So is each of the Mac's clones, oh-my-zsh
  and its two plugins, by the full commit SHA that mise checks it out at
  ([zsh](../macos/roles/zsh/README.md#the-clones)). Go is checked against the
  SHA-256 its publisher lists, on the Mac too
  ([mise.macos.lock](../macos/mise/mise.macos.lock)), and the postgres-language-server and
  docker-language-server against theirs. mise's lock holds each for its platform
  ([mise.guest.lock](../ubuntu/guest/mise/mise.guest.lock)).
  gopls and Delve are built through Go's checksum database, as mise's `go:` tools
  ([devtools](../ubuntu/roles/devtools/README.md#the-tools)). roslyn-language-server is pinned by version alone
  ([units.toml.tmpl](../ubuntu/roles/devtools/units.toml.tmpl)). The npm servers are checked against the sha512
  `integrity` of their lockfile ([devtools](../ubuntu/roles/devtools/README.md#the-npm-servers)). Docker's
  apt key is checked against its fingerprint, which mise's source pins
  ([mise.guest.toml](../ubuntu/roles/docker/mise.guest.toml)). So is Zabbly's key for Incus,
  by the fingerprint that [mise.host.toml](../ubuntu/roles/incus/mise.host.toml) pins. So is the guests' image,
  Ubuntu's root tarball, by the SHA-256 that its release lists for each architecture
  ([image.go](../ubuntu/roles/incus/image.go)). So is the git-core PPA's key,
  against Launchpad's fingerprint of it ([mise.guest.toml](../ubuntu/roles/base/mise.guest.toml),
  [base](../ubuntu/roles/base/README.md#git)). Zed's other
  downloads on ws are outside the rule
  ([zed](../ubuntu/roles/zed/README.md#zeds-downloads-outside-the-sha-256-rule)).
- The secrets' fetch trusts github.com by the one ed25519 key that ws pins
  ([github_known_hosts](../ubuntu/roles/secrets/github_known_hosts)). A test checks it
  against the fingerprint that GitHub publishes
  ([pins_test.go](../ubuntu/roles/secrets/pins_test.go),
  [The fetch](../ubuntu/roles/secrets/README.md#the-fetch)).
- foundryctl forges SigNoz's compose files, `ubuntu/roles/signoz/deployment/` and
  `casting.yaml.lock`, from `casting.yaml`. Change the casting and forge again, never the files by
  hand: CI's `signoz` job fails where they differ
  ([The compose files](../ubuntu/roles/signoz/README.md#the-compose-files)).
- Secrets are never in this repository ([By hand](../README.md#by-hand)).
- ws is public. No word of the local list, `~/.config/ws/forbidden-words`, goes into it. That
  holds for its files, the names of files and branches, the commit messages, and the text of pull
  requests and issues. The hooks in `.githooks` refuse a commit that holds one. They name an added
  line by its file and number, never its text, but print a file name that holds one whole. Never
  print the list or a word of it, and never pass `--no-verify`
  ([The hooks](../.github/workflows/README.md#the-hooks)).
- `make lint` fails on any finding, warnings included, in any line of any file
  ([The gates](../.tools/README.md); #47).
- Fix a finding, or justify it in place with its linter's own mechanism and a reason:
  `//nolint:NAME // reason`, `# yamllint disable-line rule:NAME` or
  `# shellcheck disable=SCNNNN`, each with the reason beside it. Never lower a severity, add an
  exclusion or keep a baseline: every file passes every gate outright. The linters' configs in
  `.tools/lint/`, yamllint's, golangci-lint's, Vale's, rumdl's, tombi's and StyLua's, are never weakened.
- ws's layers import downward: cobra lives in `cmd`, the `cli` packages and `.tools/`;
  `engine/core` imports itself alone; a role imports `engine/core`, `internal` and its own
  side, but not that side's `cli`. The Mac imports nothing of Ubuntu's side.
  depguard's lists in [golangci.yml](../.tools/lint/golangci.yml) fail an import across them.
- The caps are those of the
  [writing](https://github.com/zadykian/toolkit/blob/main/skills/writing/SKILL.md#the-caps-and-vale)
  skill, which `make sizecheck`, `make vale` and `make rumdl` check.

## Commits

Commits and pull requests follow the
[atomic-commits](https://github.com/zadykian/toolkit/blob/main/skills/atomic-commits/SKILL.md)
skill. Commits are signed, as the ruleset on `main` requires.

## Where else to look

- `.claude/rules/`, loaded with the files they cover. [ubuntu.md](rules/ubuntu.md) has the
  roles' invariants, [mac.md](rules/mac.md) the ws CLI's and
  [editors.md](rules/editors.md) what Zed's files and Fresh's share.
  [docs.md](rules/docs.md) says what each document holds. [writing.md](rules/writing.md)
  has ws's facts beside the `writing` skill.
- Plugins of the `zadykian` marketplace, [zadykian/toolkit](https://github.com/zadykian/toolkit),
  which `.claude/settings.json` enables: `flow`, with `/flow:issue`, and `/flow:land`, run by
  hand only; `docs`, with `/docs:trim`; `lint`, with `/lint:split-file`; `lean`, with
  `/lean:cost`, run by hand only; and `design`, whose skills sharpen a design before code
  ([design](https://github.com/zadykian/toolkit/blob/main/docs/design.md)). `flow` links and lands a stack with gh's extension,
  `gh extension install github/gh-stack`. The marketplace's `ref` names the release that the
  workflows' actions pin ([The rules](../.github/workflows/README.md#the-rules)).
- Hooks: the lint plugin's, which run `make lint-file` on each file Claude edits and
  `make lint-tree` as a turn ends, on a tree the maintainer vouches for. The gate code includes
  what [.claude/lint-gates](lint-gates) lists
  ([Claude's hooks](../.github/workflows/README.md#claudes-hooks)). The
  [lean plugin's](https://github.com/zadykian/toolkit/blob/main/docs/lean.md) deny a workflow
  agent with no tier, and a foreground poll that no timeout bounds.
