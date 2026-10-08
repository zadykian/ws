# The gates of `make lint`, the one gate, which CI's lint job runs, and the maintainer before a
# push. Any finding fails a gate, warnings included; make -k lint runs every gate past a failure.
# Each gate is a target of its own, and those that check files take FILES, every file of their kind
# when it is empty: make vale FILES=README.md.
FILES =
# The ref after which tools/commits checks the commits' messages, up to HEAD.
BASE = origin/main
# The Python of the lint tools' venv: 3.12 or newer, as ansible-core 2.20 needs.
PYTHON = python3

# The Python tools, at the pins of .github/lint-requirements.txt, and the collections of
# requirements.yml, which ansible-lint reads offline (.ansible-lint). Each is installed again where
# it is older than the file it comes from. tools/run downloads the other tools into .cache/tools.
VENV = .cache/venv
COLLECTIONS = .cache/collections
PIP_STAMP = $(VENV)/.installed
GALAXY_STAMP = $(COLLECTIONS)/.installed
export ANSIBLE_COLLECTIONS_PATH = $(CURDIR)/$(COLLECTIONS)

.PHONY: lint shell yamllint ansible-lint ruff sizecheck vale lychee rumdl actionlint zizmor \
	gitleaks commits

lint: shell yamllint ansible-lint ruff sizecheck vale lychee rumdl actionlint zizmor gitleaks \
	commits

$(PIP_STAMP): .github/lint-requirements.txt
	@$(PYTHON) -c 'import sys; sys.exit(sys.version_info < (3, 12))' || \
		{ echo 'make: the Python tools need Python 3.12 or newer, as ansible-core does'; \
		exit 1; }
	rm -rf $(VENV)
	$(PYTHON) -m venv $(VENV)
	$(VENV)/bin/pip install --quiet --disable-pip-version-check -r .github/lint-requirements.txt
	touch $@

$(GALAXY_STAMP): requirements.yml $(PIP_STAMP)
	rm -rf $(COLLECTIONS)
	$(VENV)/bin/ansible-galaxy collection install -r requirements.yml -p $(COLLECTIONS)
	touch $@

# ShellCheck and shfmt on the shell scripts, found by extension or shebang (tools/files).
shell:
	@set -ef; files=$$(tools/files sh $(FILES)); [ -z "$$files" ] || { \
		tools/run shellcheck $$files && tools/run shfmt -d -i 4 $$files; }

# yamllint finds the YAML files itself, and leaves out those .yamllint ignores.
yamllint: $(PIP_STAMP)
	@set -ef; if [ -z "$(FILES)" ]; then files=.; else files=$$(tools/files yaml $(FILES)); fi; \
		[ -z "$$files" ] || $(VENV)/bin/yamllint --strict $$files

# ansible-lint runs ansible-playbook's syntax check, so the venv's bin goes first on the PATH.
ansible-lint: $(PIP_STAMP) $(GALAXY_STAMP)
	PATH="$(CURDIR)/$(VENV)/bin:$$PATH" $(VENV)/bin/ansible-lint

# ruff's lint and format checks, as ruff.toml sets them.
ruff: $(PIP_STAMP)
	@set -ef; if [ -z "$(FILES)" ]; then files=.; else files=$$(tools/files py $(FILES)); fi; \
		[ -z "$$files" ] || { $(VENV)/bin/ruff check $$files && \
		$(VENV)/bin/ruff format --check $$files; }

# The size caps: 300 lines a file.
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

# Every commit of the history. --redact keeps a secret it finds out of the log, which is public.
gitleaks:
	tools/run gitleaks git --no-banner --redact --verbose .

# The headers and bodies of the commits from BASE to HEAD.
commits:
	tools/commits $(BASE)
