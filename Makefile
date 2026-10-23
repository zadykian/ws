# The gates of `make lint`, the one gate, which CI's lint job and Claude's Stop hook run, and the
# maintainer before a push. Any finding fails a gate, warnings included; make -k lint runs every
# gate past a failure.
# Each gate is a target of its own, and those that check files take FILES, every file of their kind
# when it is empty: make vale FILES=README.md.
FILES =
# The ref after which tools/commits checks the commits' messages, up to HEAD.
BASE = origin/main
# The commits gitleaks scans, as git log's options, which its --log-opts takes: every commit of
# the history when empty.
LOG_OPTS =
# The Python of the lint tools' venv: 3.12 to 3.14, the Pythons that ansible-core 2.20 supports
# and that the wheels of .github/lint-requirements.txt cover.
PYTHON = python3

# The Python tools, at the pins and SHA-256 sums of .github/lint-requirements.txt, and the
# collections of requirements.yml, which ansible-lint reads offline (.ansible-lint). Each is
# installed again where a file it comes from is newer. tools/run downloads the other tools into
# .cache/tools.
VENV = .cache/venv
COLLECTIONS = .cache/collections
PIP_STAMP = $(VENV)/.installed
GALAXY_STAMP = $(COLLECTIONS)/.installed
export ANSIBLE_COLLECTIONS_PATH = $(CURDIR)/$(COLLECTIONS)

.PHONY: lint shell yamllint ansible-lint ruff sizecheck vale lychee rumdl actionlint zizmor \
	gitleaks commits lint-file lint-tree

lint: shell yamllint ansible-lint ruff sizecheck vale lychee rumdl actionlint zizmor gitleaks \
	commits

# Nothing here runs uv, which writes lint-requirements.txt from lint-requirements.in, so the install
# stops where the .txt lacks a pin of the .in, rather than keep the old pin unseen. pip takes a file
# only where the .txt lists its SHA-256. So a file PyPI adds to a release later never runs: pip
# takes a listed one, or fails where none fits. --only-binary takes wheels alone, so no package's
# build runs here.
$(PIP_STAMP): .github/lint-requirements.in .github/lint-requirements.txt
	@$(PYTHON) -c 'import sys; sys.exit(not (3, 12) <= sys.version_info[:2] <= (3, 14))' || \
		{ echo 'make: the Python tools need Python 3.12 to 3.14, as ansible-core 2.20 supports'; \
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

$(GALAXY_STAMP): requirements.yml $(PIP_STAMP)
	rm -rf $(COLLECTIONS)
	$(VENV)/bin/ansible-galaxy collection install -r requirements.yml -p $(COLLECTIONS)
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

# ansible-lint runs ansible-playbook's syntax check, so the venv's bin goes first on the PATH.
# Without FILES, it finds the files itself, and leaves out those .ansible-lint excludes.
ansible-lint: $(PIP_STAMP) $(GALAXY_STAMP)
	@set -ef; files=; if [ -n "$(FILES)" ]; then files=$$(tools/files yaml $(FILES)); \
		[ -n "$$files" ] || exit 0; fi; \
		PATH="$(CURDIR)/$(VENV)/bin:$$PATH" $(VENV)/bin/ansible-lint $$files

# ruff's lint and format checks, as ruff.toml sets them.
ruff: $(PIP_STAMP)
	@set -ef; if [ -z "$(FILES)" ]; then files=.; else files=$$(tools/files py $(FILES)); fi; \
		[ -z "$$files" ] || { $(VENV)/bin/ruff check $$files && \
		$(VENV)/bin/ruff format --check $$files; }

# The size caps: 300 lines a file, and 150 the README.
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

# The headers and bodies of the commits from BASE to HEAD.
commits:
	tools/commits $(BASE)

# The gates of each file of FILES, by its kind, as the lint plugin runs them after Claude edits a
# file (docs/checks.md). A workflow takes actionlint and zizmor too. A file of no kind passes, as
# each gate passes over the files not of its kind.
lint-file:
	@status=0; for file in $(FILES); do \
		case $$file in \
		*.md) gates='vale lychee rumdl sizecheck' ;; \
		.github/workflows/*.yml | .github/workflows/*.yaml) \
			gates='yamllint ansible-lint actionlint zizmor sizecheck' ;; \
		*.yml | *.yaml) gates='yamllint ansible-lint sizecheck' ;; \
		*.py) gates='ruff sizecheck' ;; \
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
