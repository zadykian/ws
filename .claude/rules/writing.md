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
  - "mac/ws-agent"
  - "mac/ws-agent-pin"
  - "tools/*"
---

# Writing

ws's facts beside the
[writing](https://github.com/zadykian/toolkit/blob/main/skills/writing/SKILL.md) skill, which
holds the policy for comments and docs. They cover the Markdown, and the `#` comments of the
YAML, the shell scripts, the Python and the Jinja templates.

- A task's `name` says what it does. The comment above it says why, and what a dry run or a
  container does there where that differs.
- How a role works goes in its comments, what it sets up in its page in `docs/`, and how to run
  the playbook in the README.
- No word of the local list, in prose, code or names (CLAUDE.md).
