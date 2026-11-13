# CLAUDE.md

ws sets up the maintainer's development server from code: `ws server ubuntu apply`, which runs on
the machine itself, each role as Go steps. Its roles set up packages, SSH by key and a firewall, the
links to the Mac, an IKEv2 VPN where a profile is given, and the fans. Then the shell, tmux,
lazygit, Fresh, Docker, claude, SigNoz with its collector, and Zed's server. `roles/client/macos/`
sets up the Mac that works on the server. The rest is the docs, the lint gates' tools and CI.

## Layout

- `roles/server/ubuntu/NAME`: a role per directory, as Go steps, with its page, `docs/NAME.md`,
  and a line in the README. `roles/server/ubuntu` lists the roles in the order they run, and
  `roles/server/ubuntu/values` types the machine's values: `/etc/ws/host_vars.yml`, on the machine
  alone, such as the cable's port ([The direct link](roles/server/ubuntu/network/README.md#the-direct-link)). Each role
  holds its pins, and embeds its templates and files. ws-fans, the fans' service, is
  `ws server ubuntu fans`, in `roles/server/ubuntu/fans/daemon`.
- `go.mod`, `go.sum` and `.golangci.yml`: the Go module `github.com/zadykian/ws`, at the root.
  `main.go` runs `cmd`, the command. `engine/core` holds `step`,
  `effect`, with `effect/effecttest`, `effect/local` and `effect/units`, and `resource`, the resources
  the steps share, which are public: another module builds steps on them.
- `GLOSSARY.md`: the words of the server's side and of the Mac's, such as the account and the
  home file.
- `install.sh`: installs ws's binary from a release ([Releases](docs/releases.md)).
- `roles/client/macos/`: the Mac's side, the files that the ws CLI embeds ([The Mac](docs/mac.md)). These are
  `ws.ghostty`, Ghostty's settings, and `zed/`, Zed's settings
  and keymap. `zshrc` is the zsh hook that turns off the modes that a drop leaves on.
- `docs/`: a page per role; `releases.md`; `mac.md`, with `mac-git.md`, `mac-wizard.md`,
  `mac-targets.md`, `mac-ghostty.md`, `mac-zed.md` and `mac-secrets.md`; `server-wizard.md`;
  `router.md` and `checks.md`; `tui.md`, the terminal UI, with `tui-configure.md` and `tui-wizard.md`.
- `tools/`: the gates' tools. `run` pins the binaries, `files` lists a gate's files and `sizecheck`
  caps them. `build` builds ws for a target, and `vale-comments` runs Vale on files' comments.
- `.githooks/`: the hooks that check each commit for the words of the local list and for secrets.
- `.github/workflows/`: `ci.yml`, with the jobs `lint`, `playbook`, `rootless`, `install` and
  `signoz`;
  `mac.yml`, with `mac`, whose cases are `.github/mac/NN-*.sh`; `mac-cld.yml`, with `mac-cld`;
  `mac-wizard.yml`, with `mac-wizard`, whose cases are `.github/mac-wizard/NN-*.sh`; `rules.yml`,
  the checks of zadykian/toolkit on a pull request; and `issues.yml`, its check of an issue. The
  toolkit's checks warn alone. `words.yml`, with `words`, checks a pull request and an issue for the
  list's words ([The `words` job](docs/checks.md#the-words-job)).
  `release.yml`, on a tag `v*`, has `check`, `ci`, `mac`, `build` and `publish`.
- `.github/pull_request_template.md` and `.github/ISSUE_TEMPLATE/`: the toolkit's templates, which
  its `tools/pin` writes, never edited here ([The rules](docs/checks.md#the-rules)).

## Commands

```sh
make lint                         # every gate, as CI's lint job runs it; -k goes past a failure
make vale FILES=README.md         # one gate alone; the gates that check files take FILES
make shell                        # ShellCheck and shfmt -d -i 4 on every shell script
make gotest                       # the Go gates run at the root, the module; so do govet and gofmt
```

The gates are `shell`, `yamllint`, `sizecheck`, `vale`, `lychee`, `rumdl`, `actionlint`, `zizmor`,
`gitleaks`, `gofmt`, `govet`, `golangci-lint`, `govulncheck` and `gotest`
([The gates](docs/gates.md)). `make lint` needs Python 3.12 to 3.14, on Linux or on macOS on arm64.
It installs yamllint into `.cache`, and `tools/run` downloads the rest there.

The tests are CI's jobs ([Checks](docs/checks.md)). `playbook` builds ws and runs `ws server ubuntu
apply` twice in an Ubuntu 26.04 container and checks the config that Fresh reads, `signoz` forges
SigNoz's compose files again, and `install` tries `install.sh`. `rootless` runs the rootless daemon
and the firewall on a booted runner ([The rootless job](docs/checks-rootless.md)). `mac` sets up a
Mac on GitHub's macOS runner, and `mac-wizard` runs the Mac's wizard on another. `mac-cld` sets up
Ghostty and cld on a third ([Ghostty and cld on the Mac](docs/checks-mac-cld.md)). A run of `ws
server ubuntu apply` sets up the machine that runs it, as below.

The hooks check a commit for the local list's words. This checks any other text, such as a pull
request's body, as they do, and prints 0 where it holds none:

```sh
LC_ALL=C tr _ ' ' <FILE | grep -ciwF -f <(sed -e 'y/_/ /' -e 's/^[[:space:]]*//' \
    -e 's/[[:space:]]*$//' -e '/^$/d' -e '/^#/d' ~/.config/ws/forbidden-words)
```

## How changes land

- Pull requests are stacked, each on the one below it. CI runs `lint` and `signoz` on each,
  whatever its base, and `playbook`, `mac`, `mac-cld` and `mac-wizard` on the stack's top one alone
  ([Checks](docs/checks.md#checks)).
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
  container ([The `playbook` job](docs/checks.md#the-playbook-job)).
- A second run changes nothing: CI's `playbook` job runs it twice and fails where the second run
  reports a change ([ci.yml](.github/workflows/ci.yml)).
- A step that needs a booted machine's systemd checks `systemd.Booted`, and skips without it, as in
  CI's container ([The `playbook` job](docs/checks.md#the-playbook-job)).
- `release.yml` checks a release's files and notes for the local list's words before it publishes
  them. The list is the secret `FORBIDDEN_WORDS`, and the job fails on an empty one
  ([Cutting a release](docs/releases.md#cutting-a-release)).
- A pinned download is checked against its SHA-256 before use: ikeward's
  binary ([binary.go](roles/server/ubuntu/ikeward/binary.go)), otelcol's `.deb`
  ([download.go](roles/server/ubuntu/otelcol/download.go)) and Docker's apt key
  ([repository.go](roles/server/ubuntu/docker/repository.go)). So is Fresh's `.deb`
  ([download.go](roles/server/ubuntu/fresh/download.go)), and so are foundryctl
  ([ci.yml](.github/workflows/ci.yml)) and the gates' tools ([tools/run](tools/run)). So is ws's
  binary, against its release's `ws.sha256` ([install.sh](install.sh),
  [update.go](engine/core/update/update.go)), and so are yamllint
  ([lint-requirements.in](.github/lint-requirements.in)) and Homebrew's `.pkg`, with its signer
  ([homebrew.go](roles/client/macos/stages/tools/homebrew.go)). Go is checked against the
  SHA-256 its publisher lists ([devtools](roles/server/ubuntu/devtools/README.md)). The postgres-language-server and
  docker-language-server are checked against their SHA-256
  ([binaries.go](roles/server/ubuntu/devtools/binaries.go)). gopls and Delve are built through Go's
  checksum database. roslyn-language-server is pinned by version alone
  ([pins.go](roles/server/ubuntu/devtools/pins.go)). The npm servers are checked against the sha512
  `integrity` of their lockfile ([devtools](roles/server/ubuntu/devtools/README.md#the-npm-servers)). Zed's other
  downloads on ws are outside the rule
  ([zed](roles/server/ubuntu/zed/README.md#zeds-downloads-outside-the-sha-256-rule)).
- foundryctl forges SigNoz's compose files, `roles/server/ubuntu/signoz/deployment/` and
  `casting.yaml.lock`, from `casting.yaml`. Change the casting and forge again, never the files by
  hand: CI's `signoz` job fails where they differ
  ([The compose files](roles/server/ubuntu/signoz/README.md#the-compose-files)).
- Secrets are never in this repository ([By hand](README.md#by-hand)).
- ws is public. No word of the local list, `~/.config/ws/forbidden-words`, goes into it. That
  holds for its files, the names of files and branches, the commit messages, and the text of pull
  requests and issues. The hooks in `.githooks` refuse a commit that holds one. They name an added
  line by its file and number, never its text, but print a file name that holds one whole. Never
  print the list or a word of it, and never pass `--no-verify`
  ([The hooks](docs/checks.md#the-hooks)).
- `make lint` fails on any finding, warnings included, in any line of any file
  ([The gates](docs/gates.md); #47).
- Fix a finding, or justify it in place with its linter's own mechanism and a reason:
  `//nolint:NAME // reason`, `# yamllint disable-line rule:NAME` or
  `# shellcheck disable=SCNNNN`, each with the reason beside it. Never lower a severity, add an
  exclusion or keep a baseline: every file passes every gate outright, and `.yamllint`,
  `.golangci.yml` and `.vale.ini` are never weakened.
- ws's layers import downward: cobra lives in `cmd`, the `cli` packages and `tools/`;
  `engine/core` imports itself alone; a role imports `engine/core`, `roles/internal` and its own
  side; the Mac imports the server's `values`, `secrets`, `session`, `stages` and `cli` alone.
  depguard's lists in [.golangci.yml](.golangci.yml) fail an import across them.
- The caps are those of the
  [writing](https://github.com/zadykian/toolkit/blob/main/skills/writing/SKILL.md#the-caps-and-vale)
  skill, which `make sizecheck`, `make vale` and `make rumdl` check.

## Commits

Commits and pull requests follow the
[atomic-commits](https://github.com/zadykian/toolkit/blob/main/skills/atomic-commits/SKILL.md)
skill. Commits are signed, as the ruleset on `main` requires.

## Where else to look

- `.claude/rules/`, loaded with the files they cover. [server.md](.claude/rules/server.md) has the
  roles' invariants, [mac.md](.claude/rules/mac.md) the ws CLI's and
  [editors.md](.claude/rules/editors.md) what Zed's files and Fresh's share.
  [docs.md](.claude/rules/docs.md) says what each document holds. [writing.md](.claude/rules/writing.md)
  has ws's facts beside the `writing` skill.
- Plugins of the `zadykian` marketplace, [zadykian/toolkit](https://github.com/zadykian/toolkit),
  which `.claude/settings.json` enables: `flow`, with `/flow:issue`, and `/flow:land`, run by
  hand only; `docs`, with `/docs:trim`; `lint`, with `/lint:split-file`; `lean`, with
  `/lean:cost`, run by hand only; and `design`, whose skills sharpen a design before code
  ([design](https://github.com/zadykian/toolkit/blob/main/docs/design.md)). `flow` links and lands a stack with gh's extension,
  `gh extension install github/gh-stack`. The marketplace's `ref` names the release that the
  workflows' actions pin ([The rules](docs/checks.md#the-rules)).
- Hooks: the lint plugin's, which run `make lint-file` on each file Claude edits and
  `make lint-tree` as a turn ends, on a tree the maintainer vouches for. The gate code includes
  what [.claude/lint-gates](.claude/lint-gates) lists
  ([Claude's hooks](docs/checks.md#claudes-hooks)). The
  [lean plugin's](https://github.com/zadykian/toolkit/blob/main/docs/lean.md) deny a workflow
  agent with no tier, and a foreground poll that no timeout bounds.
