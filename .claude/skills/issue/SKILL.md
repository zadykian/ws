---
name: issue
description: >-
  Plan, then file a GitHub issue for ws in its compact house style once the maintainer approves the
  plan. Use when asked to file, open or write up an issue, or to plan a change.
argument-hint: "[topic]"
---

# File an issue

Plan, approval, then the issue. Never branch or write code unless the maintainer asks for it.

## 1. Plan

Reply with a plan and ask for approval:

- the goal, in a sentence or two;
- the changes, with `file:line` references;
- what to check: the gates of `make lint`, and what CI's `playbook`, `signoz` and `mac` jobs run;
- open decisions, each with a recommendation;
- the pull requests, and where they go in the stack.

"Approved, create the issue" means `gh issue create`, not code.

## 2. Read the latest main

```sh
git fetch origin
git rev-parse --short origin/main      # the SHA the issue cites
git show origin/main:PATH              # read files at that SHA
```

The local checkout may lag main by several commits: cite only lines read at `origin/main`.

## 3. Write it

- **Title:** `type(scope): description`, as a commit's header in CLAUDE.md: the scope is the role,
  or `mac`, or none. The description is a short noun phrase: `feat(base): a swappiness of 10`.
- **Summary:** two to four sentences: what is missing or wrong, and what the issue adds. It ends
  "Line references are to main SHA."
- **`## Today`:** bullets with `file:line` references, the crucial facts only.
- **`## Pull requests`:** only where there are several, a checkbox per pull request with its
  header.
- **`## Proposal`:** bullets saying what changes.
- **`## Open decisions`:** only when something is undecided, each with options and a
  recommendation.
- 150 to 350 words in all, as #47, #48 and #49. Not the long style of #34 and #36.

```markdown
SUMMARY. Line references are to main SHA.

## Today

- `roles/NAME/tasks/main.yml:12-30` does X.

## Proposal

- Do Y.
```

## 4. File it

ws is public, so no word of the local list goes into the title or the body (CLAUDE.md). Write the
body to a file in the scratchpad, never in the repository, and check it and the title with the
local list's check in CLAUDE.md's Commands. Then:

```sh
gh issue create -R zadykian/ws --title 'TITLE' --body-file BODY [--label LABEL]
```

Labels: `feat` gets `enhancement`, `fix` gets `bug`, `docs` gets `documentation`, and the other
types none. Reply with the issue's URL.
