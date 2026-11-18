---
paths:
  - "**/*.md"
  - "**/*.go"
  - "**/*.yml"
  - "**/*.yaml"
  - "**/*.sh"
  - "**/*.toml"
  - "**/*.lua"
  - ".githooks/*"
  - ".tools/*"
---

# Writing

ws's facts beside the
[writing](https://github.com/zadykian/toolkit/blob/main/skills/writing/SKILL.md) skill, which
holds the policy for comments and docs. They cover the Markdown and every comment: Go's and Lua's,
and those of the YAML, the shell scripts, the templates and the configs. `make vale` checks them all
([The gates](../../.tools/README.md)).

- A unit says what it does by its name and its spec's comment. A comment says why, and what a
  plan or a container does there where that differs.
- How a role works goes in its comments, and what it sets up in its README. How to run the
  bootstrap goes in the root README and the runbooks.
- No word of the local list, in prose, code or names (CLAUDE.md).
