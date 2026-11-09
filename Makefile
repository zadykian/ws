# The gates of `make lint`, the one gate, which CI's lint job runs, the lint plugin through
# lint-tree, and the maintainer before a push. Any finding fails a gate, warnings included; make -k
# lint runs every gate past a failure.
# Each gate is a target of its own, and those that check files take FILES, every file of their kind
# when FILES is empty: make vale FILES=README.md.
FILES =
# The ref the branch starts from: lint-tree's gitleaks scans the commits after it, up to HEAD.
BASE = origin/main
# The commits gitleaks scans, as git log's options, which its --log-opts takes: every commit of
# the history when empty.
LOG_OPTS =
# The Python of yamllint's venv: 3.12 to 3.14, the Pythons that the wheels of
# .github/lint-requirements.txt cover.
PYTHON = python3

# yamllint, the one Python tool, at the pins and SHA-256 sums of .github/lint-requirements.txt. It
# is installed again where a file it comes from is newer. tools/run downloads the other tools into
# .cache/tools.
VENV = .cache/venv
PIP_STAMP = $(VENV)/.installed

# Go's tools, from tools/run, keep their caches in .cache and never fetch another toolchain. ws
# builds without cgo; gotest turns it on for the race detector alone.
export GOPATH = $(CURDIR)/.cache/go
export GOMODCACHE = $(CURDIR)/.cache/go/pkg/mod
export GOCACHE = $(CURDIR)/.cache/go-build
export GOTOOLCHAIN = local
export GOWORK = off
export CGO_ENABLED = 0

.PHONY: lint shell yamllint sizecheck vale lychee rumdl actionlint zizmor \
	gitleaks gofmt govet golangci-lint govulncheck gotest lint-file lint-tree

lint: shell yamllint sizecheck vale lychee rumdl actionlint zizmor gitleaks \
	gofmt govet golangci-lint govulncheck gotest

# Nothing here runs uv, which writes lint-requirements.txt from lint-requirements.in. So the install
# stops where the .txt lacks a pin of the .in, rather than keep the old pin unseen. pip takes a file
# only where the .txt lists its SHA-256. So a file PyPI adds to a release later never runs: pip
# takes a listed one, or fails where none fits. --only-binary takes wheels alone, so no package's
# build runs here.
$(PIP_STAMP): .github/lint-requirements.in .github/lint-requirements.txt
	@$(PYTHON) -c 'import sys; sys.exit(not (3, 12) <= sys.version_info[:2] <= (3, 14))' || \
		{ echo 'make: the Python tools need Python 3.12 to 3.14, as the lint wheels cover'; \
		exit 1; }
	@awk 'NR == FNR { if (!/^(#|$$)/) pin[$$1] = 1; next } { delete pin[$$1] } \
		END { for (p in pin) { print "make: lint-requirements.txt lacks " p \
		", which lint-requirements.in pins: run uv as the .in says"; s = 1 } exit s }' \
		.github/lint-requirements.in .github/lint-requirements.txt
	rm -rf $(VENV)
	$(PYTHON) -m venv $(VENV)
	$(VENV)/bin/pip install --quiet --disable-pip-version-check --require-hashes \
		--only-binary :all: -r .github/lint-requirements.txt
	touch $@

# ShellCheck and shfmt on the shell scripts, found by extension or shebang (tools/files). -x
# follows a script's source directives to the files outside FILES, so that a script checked alone
# sees what it sources.
shell:
	@set -ef; files=$$(tools/files sh $(FILES)); [ -z "$$files" ] || { \
		tools/run shellcheck -x $$files && tools/run shfmt -d -i 4 $$files; }

# yamllint finds the YAML files itself, and leaves out those .yamllint ignores.
yamllint: $(PIP_STAMP)
	@set -ef; if [ -z "$(FILES)" ]; then files=.; else files=$$(tools/files yaml $(FILES)); fi; \
		[ -z "$$files" ] || $(VENV)/bin/yamllint --strict $$files

# The size cap: 300 lines a file.
sizecheck:
	tools/sizecheck $(FILES)

# Vale (.vale.ini) on the Markdown files git lists, new ones included.
vale:
	@set -ef; files=$$(tools/files md $(FILES)); [ -z "$$files" ] || tools/run vale $$files

# Links to files and their headings in the Markdown files.
lychee:
	@set -ef; files=$$(tools/files md $(FILES)); [ -z "$$files" ] || \
		tools/run lychee --offline --include-fragments --no-progress $$files

# rumdl (.rumdl.toml) on the Markdown files.
rumdl:
	@set -ef; files=$$(tools/files md $(FILES)); [ -z "$$files" ] || tools/run rumdl check $$files

# The workflows' shell scripts go through the pinned ShellCheck.
actionlint:
	shellcheck="$$(tools/run -path shellcheck)" && \
		tools/run actionlint -shellcheck="$$shellcheck" -pyflakes=

# The workflows, with the online audits where GH_TOKEN is set, as in CI.
zizmor:
	tools/run zizmor .

# The commits of LOG_OPTS. --redact keeps a secret it finds out of the log, which is public.
gitleaks:
	tools/run gitleaks git --no-banner --redact --verbose \
		$(if $(LOG_OPTS),--log-opts='$(LOG_OPTS)') .

# The Go gates run at the root, the module, with the Go that tools/run pins first on the PATH.
# gofmt fails where it lists a file.
gofmt:
	@set -ef; go=$$(tools/run -path go); out=$$(git ls-files -z "*.go" | xargs -0 "$${go%/go}/gofmt" -l); \
		[ -z "$$out" ] || { echo "$$out"; echo "gofmt lists these files"; exit 1; }

govet:
	@set -ef; go=$$(tools/run -path go); "$$go" vet ./...

golangci-lint:
	@set -ef; go=$$(tools/run -path go); lint=$$(tools/run -path golangci-lint); \
		PATH="$${go%/go}:$$PATH" "$$lint" run ./...

# govulncheck is the module's tool directive, at the version go.mod pins.
govulncheck:
	@set -ef; go=$$(tools/run -path go); "$$go" tool govulncheck ./...

# The race detector needs cgo, for the tests alone. LAZYGIT_SCHEMA names lazygit's config schema,
# which the lazygit role's test holds its config to.
gotest:
	@set -ef; go=$$(tools/run -path go); schema=$$(tools/run -path lazygit-schema); \
		LAZYGIT_SCHEMA=$$schema CGO_ENABLED=1 "$$go" test -race -count=1 ./...

# The gates of each file of FILES, by its kind, as the lint plugin runs them after Claude edits a
# file (docs/checks.md). A workflow takes actionlint and zizmor too. A file of no kind passes, as
# each gate passes over the files not of its kind.
lint-file:
	@status=0; for file in $(FILES); do \
		case $$file in \
		*.md) gates='vale lychee rumdl sizecheck' ;; \
		.github/workflows/*.yml | .github/workflows/*.yaml) \
			gates='yamllint actionlint zizmor sizecheck' ;; \
		*.yml | *.yaml) gates='yamllint sizecheck' ;; \
		*.go | go.mod | go.sum) gates='gofmt govet golangci-lint govulncheck gotest sizecheck' ;; \
		*.j2) gates=sizecheck ;; \
		*) gates='shell sizecheck' ;; \
		esac; \
		$(MAKE) --no-print-directory $$gates FILES="$$file" || status=1; \
	done; exit $$status

# Every gate, as the lint plugin runs it as Claude ends its turn, with BASE at origin/main's ref.
# gitleaks scans the branch's own commits alone, as a secret in a stash or in another branch,
# which every worktree shares, is none of the turn's.
lint-tree: LOG_OPTS = $(BASE)..HEAD
lint-tree: lint
