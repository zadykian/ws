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

How comments and docs are written: the Markdown, and the `#` comments of the YAML, the shell
scripts, the Python and the Jinja templates. `make sizecheck`, `make vale` and `make rumdl` check
the measurable part, each warning counting as an error; review checks the rest.

## What to write

- A comment gives a block's purpose in one or two sentences, or a reason the code does not show,
  with the finding behind it. It never narrates the code.
- A task's `name` says what it does. The comment above it says why, and what a dry run or a
  container does there where that differs.
- Each fact lives in one place. How a role works goes in its comments, what it sets up in its page
  in `docs/`, and how to run the playbook in the README. Other places link to it.
- A doc says what the code leaves out: what a role sets up and leaves by hand, its variables, and
  its caveats.
- No word of the local list, in prose, code or names (CLAUDE.md).

## The caps

- Sentences of 30 words or fewer. Vale counts the words outside code spans, and reads the Markdown
  alone, so a comment keeps to the rule by review.
- Files of 300 lines or fewer, for Markdown, YAML, shell, Python and Jinja templates; the README of
  150 or fewer.
- Prose is wrapped at 100 columns: the Markdown, as rumdl checks, and the comments, whose lines
  yamllint and ruff hold to 100 too.
- Vale refuses wordy phrases (`write-good.TooWordy`), such as `however`, `minimum`, `multiple`,
  `in order to` or `it is`, and needless variants (`proselint.Needless`). Comments avoid them too.
