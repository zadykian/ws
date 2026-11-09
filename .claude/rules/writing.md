---
paths:
  - "**/*.md"
  - "**/*.go"
  - "**/*.yml"
  - "**/*.yaml"
  - "**/*.sh"
  - ".githooks/*"
  - "tools/*"
---

# Writing

ws's facts beside the
[writing](https://github.com/zadykian/toolkit/blob/main/skills/writing/SKILL.md) skill, which
holds the policy for comments and docs. They cover the Markdown and every comment: Go's, and
those of the YAML, the shell scripts, the templates and the configs. `make vale` checks them all
([The gates](../../docs/gates.md)).

- A step's `Name` says what it does. The comment above it says why, and what a dry run or a
  container does there where that differs.
- How a role works goes in its comments, what it sets up in its page in `docs/`, and how to run
  `ws server apply` in the README.
- No word of the local list, in prose, code or names (CLAUDE.md).
