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
- `yamllint`: `yamllint --strict`, as `.yamllint` sets it, on the YAML files it finds itself.
  Those include `host_vars/localhost.yml`, which git ignores, so that file starts with `---`
  ([The direct link](network.md#the-direct-link));
- `ansible-lint`: its production profile, strict and offline (`.ansible-lint`);
- `ruff`: `ruff check` and `ruff format --check` on the Python files, as `ruff.toml` sets them;
- `sizecheck`: 300 lines a file, for Markdown, YAML, shell, Python and Jinja templates;
- `vale`: Vale's three rules (`.vale.ini`) on the Markdown files: sentences of 30 words or fewer,
  no wordy phrases and no needless variants;
- `lychee`: the links to files and their headings in the Markdown files, offline;
- `rumdl`: rumdl's rules on the Markdown files, with lines of 100 columns but in code blocks and
  tables (`.rumdl.toml`);
- `actionlint` and `zizmor`: the workflows, zizmor with its online audits where `GH_TOKEN` is set;
- `gitleaks`: every commit, for secrets, or those that `LOG_OPTS` picks with git log's options.

`tools/sizecheck` leaves out the files nobody writes here. Those are foundryctl's compose files and
lock, which yamllint and ansible-lint leave out too, and Vale's vendored styles. Data files, such
as TOML, JSON and conf files, are of no kind it caps. `.editorconfig` sets the files' encoding,
line ends and indents for editors.

`tools/run` downloads the other tools into `.cache/tools`, each pinned by version and SHA-256. make
installs the Python tools of `.github/lint-requirements.txt` into `.cache/venv`, with a Python of
3.12 to 3.14, which ansible-core 2.20 supports. That file pins each package with the SHA-256 of
each of its files. pip takes only the wheels it lists, and builds no package. So the tools install
where each package has a wheel: on Linux, on amd64 or arm64, and on macOS on arm64, as cryptography
has none for macOS on amd64. uv writes the file from `.github/lint-requirements.in`, which says
how, and make stops where the file lacks a pin of the `.in`. make installs the collections of
`requirements.yml` into `.cache/collections`, as ansible-lint runs offline. It installs each again
where a file it comes from is newer.

[writing]: https://github.com/zadykian/toolkit/blob/main/skills/writing/SKILL.md#the-caps-and-vale
