# The gates

`make lint` is the one gate: CI's lint job runs it, the lint plugin through `make lint-tree` as
Claude ends its turn, and the maintainer before a push ([Claude's hooks](checks.md#claudes-hooks)).
Any finding fails it, warnings included. A finding is fixed, or justified in place with its linter's
own inline mechanism and a reason. A Vale alert is always fixed, by a rewrite of the sentence, as
[the writing skill][writing] allows no exception to its rules. No gate lowers a severity or keeps a
baseline, and none leaves out a file written here. `make -k lint` runs every gate past a failure.
Each gate is a target of its own, and those that
check files take `FILES`, as in `make vale FILES=README.md`:

- `shell`: ShellCheck, which follows a script's sources with `-x`, and `shfmt -d -i 4` on every
  shell script, found by extension or shebang;
- `yamllint`: `yamllint --strict`, as `.yamllint` sets it, on the YAML files it finds itself;
- `sizecheck`: 300 lines a file, for Markdown, YAML, shell and Go;
- `vale`: Vale's three rules (`.vale.ini`): sentences of 30 words or fewer, no wordy phrases and
  no needless variants. It checks the Markdown files, and the comments of every other file written
  here. Vale would read such a file's code too. So `tools/vale-comments` gives Vale the
  comments alone, mirrored as Markdown, and maps each alert back to its line and column. It skips
  shebangs and linter directives, though not a directive's reason. Of `#` comments, it reads full
  lines alone. A doc's leading declared name is code to it;
- `lychee`: the links to files and their headings in the Markdown files, offline;
- `rumdl`: rumdl's rules on the Markdown files, with lines of 100 columns but in code blocks and
  tables (`.rumdl.toml`);
- `actionlint` and `zizmor`: the workflows, zizmor with its online audits where `GH_TOKEN` is set;
- `gitleaks`: every commit, for secrets, or those that `LOG_OPTS` picks with git log's options;
- `gofmt`, `govet`, `golangci-lint`, `govulncheck` and `gotest`: the Go module at the root.
  `gofmt` fails where `gofmt -l` lists a file, and `golangci-lint` runs a strict set of linters
  (`.golangci.yml`). Its depguard lists, one a layer, fail an import across ws's layers and say
  why. A finding is justified with `//nolint:NAME // reason` alone.
  `govulncheck` is the module's `tool` directive, so go.mod pins it. `gotest` runs
  `go test -race -count=1 ./...`, and cgo is on for the race detector alone, as ws builds with
  `CGO_ENABLED=0`. `LAZYGIT_SCHEMA` names lazygit's config schema, for
  [the lazygit role's test](lazygit.md#the-config).

`tools/sizecheck` leaves out the files nobody writes here. Those are foundryctl's compose files and
lock, which yamllint leaves out too, and Vale's vendored styles. Data files, such
as TOML, JSON and conf files, are of no kind it caps. `.editorconfig` sets the files' encoding,
line ends and indents for editors.

`tools/run` downloads the other tools into `.cache/tools`, each pinned by version and SHA-256,
and lazygit's config schema too, a file to read.
Go comes from go.dev, with the four sums of its listing, and golangci-lint from its release. The
Go gates keep GOPATH and the caches in `.cache`, and `GOTOOLCHAIN=local` stops a download of
another Go.

make installs yamllint, the one Python tool, with its two dependencies, into `.cache/venv`. It
takes them from `.github/lint-requirements.txt`, with a Python of 3.12 to 3.14. That file pins
each package with the SHA-256 of each of its files. pip takes only the wheels it lists, and builds
no package. uv writes the file from `.github/lint-requirements.in`, which says how, and make stops
where the file lacks a pin of the `.in`. make installs the tools again where a file they come from
is newer.

[writing]: https://github.com/zadykian/toolkit/blob/main/skills/writing/SKILL.md#the-caps-and-vale
