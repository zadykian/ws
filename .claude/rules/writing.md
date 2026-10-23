---
paths:
  - "**/*.md"
  - "**/*.yml"
  - "**/*.yaml"
  - "**/*.sh"
  - "**/*.py"
  - "**/*.j2"
  - ".githooks/*"
  - "mac/ws-tunnel"
  - "tools/*"
---

# Writing

What ws adds to the `writing` skill, which holds the shared policy for comments and docs. This
file wins where they differ. It covers the Markdown, and the `#` comments of the YAML, the shell
scripts, the Python and the Jinja templates. `make sizecheck`, `make vale` and `make rumdl` check
the measurable part, each warning counting as an error; review checks the rest.

- A task's `name` says what it does. The comment above it says why, and what a dry run or a
  container does there where that differs.
- How a role works goes in its comments, what it sets up in its page in `docs/`, and how to run
  the playbook in the README.
- A doc says what the code leaves out: what a role sets up and leaves by hand, its variables, and
  its caveats.
- No word of the local list, in prose, code or names (CLAUDE.md).
- The caps cover Markdown, YAML, shell, Python and Jinja templates, and the README is 150 lines or
  fewer.
- Vale reads the Markdown alone, so a comment keeps to its rules by review. yamllint and ruff hold
  a comment's lines to 100 columns, as rumdl holds the Markdown's.
