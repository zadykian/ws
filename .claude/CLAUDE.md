# CLAUDE.md

ws sets up the maintainer's development server from code: `ws server ubuntu apply`, which runs on
the machine itself, each role as Go steps. Its roles set up packages, SSH by key and a firewall, the
links to the Mac, an IKEv2 VPN where a profile is given, and the fans. Then the shell, git,
tmux, lazygit, Fresh, Docker, claude, SigNoz with its collector, and Zed's server. `roles/client/macos/`
sets up the Mac that works on the server. The rest is the docs, the lint gates' tools and CI.

## Layout

- `roles/server/ubuntu/NAME`: a role per directory, as Go steps, with its page, `README.md`,
  and a line in the root README. `roles/server/ubuntu` lists the roles in the order they run, and
  `roles/server/ubuntu/values` types the machine's values: `/etc/ws/values.yml`, on the machine
  alone, such as the cable's port ([The cable](../roles/server/ubuntu/cable/README.md#the-cable)). Each role
  holds its pins, and embeds its templates and files. ws-fans, the fans' service, is
  `ws server ubuntu fans`, in `roles/server/ubuntu/fans/daemon`.
- `go.mod` and `go.sum`: the Go module `github.com/zadykian/ws`, at the root.
  `main.go` runs `cmd`, which composes the root and the exit codes with the two sides'
  commands. `engine/cli` builds the root and the exit codes. `engine/core` holds `step`,
  `effect`, with `effect/effecttest`, `effect/local` and `effect/units`, and `resource`, with
  `resource/resourcetest`, the resources the steps share, and `kind`, the kinds that a role's
  `units.toml` names as units ([kind](../engine/core/kind/README.md)). They are public, as the roles
  import them from outside `engine/`.
- `mise.toml` and `mise.lock`: mise's config that both sides share, its settings and the pin of
  ws, a release, with the release's SHA-256 for each target in the lock. `mise.server.toml` is
  the server's part, and `plugins/ws` the plugin that hands ws's units to `ws unit`. A role moved
  to mise holds its spec, `units.toml`, and its mise files beside its page, and `mise/conf.d`
  links its directory ([The frame](../roles/server/ubuntu/frame/README.md)). The roles' tools
  lock in `mise/mise.server.lock`.
- `GLOSSARY.md`: the words of the server's side and of the Mac's, such as the account and the
  home file.
- `install.sh`: installs mise and runs `mise bootstrap` on a release's tree
  ([Releases](../engine/core/update/README.md)).
- `roles/client/macos/`: the Mac's side ([The Mac](../roles/client/macos/README.md)). Its roles embed
  `ghostty/files/ws.ghostty`, Ghostty's settings, `zed/files/`, Zed's settings and keymap, and
  `zsh/files/zshrc`, the zsh hook that turns off the modes that a drop leaves on.
- The pages sit beside their code, as the `README.md` of a role or a package. A second page stands
  beside one where the cap keeps two apart: `zed/projects.md`, `cable/router.md`,
  `.github/workflows/rootless.md`, `mac-cld.md`,
  `mac-servers.md`, `runbook.md` of each side, and `file.md` and `state.md` of `engine/core/kind`.
  `.tools/README.md` holds the gates and `.github/workflows/README.md` CI.
- `.tools/`: the gates' tools. `run` pins the binaries, `files` lists a gate's files and `sizecheck`
  caps them. `build` builds ws for a target, `serve-build` points a checkout's pin at the tree's build,
  for CI and a trial, and `vale-comments` runs Vale on files' comments. Go's
  `./...` leaves out a dot directory, so the Go gates name the tools' packages too. `lint/` holds
  the linters' configs, which the Makefile passes by flag, but tombi's, which tombi finds as it runs
  there.
- `.githooks/`: the hooks that check each commit for the words of the local list and for secrets.
- `.claude/`: this file, the rules of `rules/`, Claude Code's `settings.json` and the lint
  plugin's `lint-gates` ([Where else to look](#where-else-to-look)).
- `.github/workflows/`: `ci.yml`, with the jobs `lint`, `playbook`, `rootless`, `install` and
  `signoz`;
  `mac.yml`, with `mac`, whose cases are `.github/mac/NN-*.sh`; `mac-cld.yml`, with `mac-cld`;
  `rules.yml`,
  the checks of zadykian/toolkit on a pull request; and `issues.yml`, its check of an issue. The
  toolkit's checks warn alone. `words.yml`, with `words`, checks a pull request and an issue for the
  list's words ([The `words` job](../.github/workflows/README.md#the-words-job)).
  `release.yml`, on a tag `v*`, has `check`, `ci`, `mac`, `build` and `publish`.
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

The tests are CI's jobs ([Checks](../.github/workflows/README.md)). `playbook` builds ws and runs `install.sh`'s
bootstrap and `ws server ubuntu apply` twice each in an Ubuntu 26.04 container, with mise's plan.
It checks the config that Fresh reads too. `signoz` forges SigNoz's compose files again, and
`install` tries `install.sh` and `ws update`. `rootless` runs the
rootless daemon and the firewall on a booted runner ([The rootless job](../.github/workflows/rootless.md)). `mac` sets up a
Mac on GitHub's macOS runner. `mac-cld` sets up
Ghostty and cld on another ([Ghostty and cld on the Mac](../.github/workflows/mac-cld.md)). A run of `ws
server ubuntu apply` sets up the machine that runs it, as below.

The hooks check a commit for the local list's words. This checks any other text, such as a pull
request's body, as they do, and prints 0 where it holds none:

```sh
LC_ALL=C tr _ ' ' <FILE | grep -ciwF -f <(sed -e 'y/_/ /' -e 's/^[[:space:]]*//' \
    -e 's/[[:space:]]*$//' -e '/^$/d' -e '/^#/d' ~/.config/ws/forbidden-words)
```

## How changes land

- Pull requests are stacked, each on the one below it. CI runs `lint` and `signoz` on each,
  whatever its base, and `playbook`, `mac` and `mac-cld` on the stack's top one alone
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

- `ws server ubuntu apply` sets up the machine that runs it. CI's `playbook` job runs it in a
  container ([The `playbook` job](../.github/workflows/README.md#the-playbook-job)).
- A second run changes nothing: CI's `playbook` job runs the bootstrap and `ws server ubuntu apply`
  twice each. It fails where the second apply reports a change, or where
  `mise bootstrap plan --detailed-exitcode` does not exit 0, before the second bootstrap and after
  both ([ci.yml](../.github/workflows/ci.yml)).
- A step that needs a booted machine's systemd checks `systemd.Booted`, and skips without it, as in
  CI's container ([The `playbook` job](../.github/workflows/README.md#the-playbook-job)).
- `release.yml` checks a release's files and notes for the local list's words before it publishes
  them. The list is the secret `FORBIDDEN_WORDS`, and the job fails on an empty one
  ([Cutting a release](../engine/core/update/README.md#cutting-a-release)).
- A pinned download is checked against its SHA-256 before use: ikeward's
  binary ([units.toml](../roles/server/ubuntu/ikeward/units.toml)) and otelcol's `.deb`
  ([units.toml](../roles/server/ubuntu/otelcol/units.toml)). So is Fresh's `.deb`
  ([units.toml](../roles/server/ubuntu/fresh/units.toml)). So is nct6687's source
  ([fans](../roles/server/ubuntu/fans/units.toml)), and so are foundryctl
  ([ci.yml](../.github/workflows/ci.yml)) and the gates' tools ([.tools/run](../.tools/run)). So is ws's
  binary: `mise.lock` holds its SHA-256 for each target, which mise checks as it installs
  ([mise.lock](../mise.lock)). `ws update` checks it against its release's `ws.sha256`
  ([update.go](../engine/core/update/update.go)), and a Mac's runbook checks it by hand
  ([runbook.md](../roles/client/macos/runbook.md)). So is mise's binary, whose SHA-256 for each
  target `install.sh` holds ([install.sh](../install.sh)). So is
  yamllint ([lint-requirements.in](../.github/lint-requirements.in)), and Homebrew's `.pkg`, with its signer
  ([runbook.md](../roles/client/macos/runbook.md)). Go is checked against the
  SHA-256 its publisher lists, and the postgres-language-server and docker-language-server against
  theirs. mise's lock holds each for its platform ([mise.server.lock](../mise/mise.server.lock)).
  gopls and Delve are built through Go's checksum database, as mise's `go:` tools
  ([devtools](../roles/server/ubuntu/devtools/README.md#the-tools)). roslyn-language-server is pinned by version alone
  ([pins.go](../roles/server/ubuntu/devtools/pins.go)). The npm servers are checked against the sha512
  `integrity` of their lockfile ([devtools](../roles/server/ubuntu/devtools/README.md#the-npm-servers)). Docker's
  apt key is checked against its fingerprint, which mise's source pins
  ([mise.server.toml](../roles/server/ubuntu/docker/mise.server.toml)). Zed's other
  downloads on ws are outside the rule
  ([zed](../roles/server/ubuntu/zed/README.md#zeds-downloads-outside-the-sha-256-rule)).
- foundryctl forges SigNoz's compose files, `roles/server/ubuntu/signoz/deployment/` and
  `casting.yaml.lock`, from `casting.yaml`. Change the casting and forge again, never the files by
  hand: CI's `signoz` job fails where they differ
  ([The compose files](../roles/server/ubuntu/signoz/README.md#the-compose-files)).
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
  `engine/core` imports itself alone; a role imports `engine/core`, `roles/internal` and its own
  side, but not that side's `cli`. The Mac imports nothing of the server's side.
  depguard's lists in [golangci.yml](../.tools/lint/golangci.yml) fail an import across them.
- The caps are those of the
  [writing](https://github.com/zadykian/toolkit/blob/main/skills/writing/SKILL.md#the-caps-and-vale)
  skill, which `make sizecheck`, `make vale` and `make rumdl` check.

## Commits

Commits and pull requests follow the
[atomic-commits](https://github.com/zadykian/toolkit/blob/main/skills/atomic-commits/SKILL.md)
skill. Commits are signed, as the ruleset on `main` requires.

## Where else to look

- `.claude/rules/`, loaded with the files they cover. [server.md](rules/server.md) has the
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
