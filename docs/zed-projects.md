# A project's `.zed/settings.json`

A project commits `.zed/settings.json` at its root. Zed on the Mac and its server on ws both read
it, above ws's own `~/.config/zed/settings.json` ([zed](zed.md)). Arrays replace the lower layers'
and objects merge key by key. Zed reads the file as JSONC, so comments are allowed.

Zed applies no project settings until the worktree is trusted
([Trust each project](zed.md#trust-each-project)). Open the project's root itself, not its parent.

## A SQL Server project

Zed's SQL language serves PostgreSQL. A T-SQL project turns its language server off and formats
with sql-formatter, which the `zed` role links to `/usr/local/bin/sql-formatter`. The settings name
the bare command, so the file stays the same on every machine.

```jsonc
{
  "languages": {
    "SQL": {
      "language_servers": ["!postgres-language-server", "..."],
      "formatter": { "external": { "command": "sql-formatter", "arguments": ["--language", "tsql"] } }
    }
  }
}
```

- sql-formatter reads `.sql-formatter.json` from the worktree root upward, as Zed runs it there.
  A file in a subdirectory is never found. Per-directory dialects need nested `.zed/settings.json`
  files.
- SQL Server gets highlighting and sql-formatter alone. No registry extension runs sqls or any
  T-SQL server (`zed-extensions/sql#2`).
- sqls v0.2.48 is its latest release. The change that replaces its SQL Server driver
  (sqls#218) is in no release. Its formatter breaks T-SQL (sqls#93).

Highlighting breaks on:

- `[bracketed]` T-SQL names (`DerekStride/tree-sitter-sql#320`);
- `$$` bodies (`DerekStride/tree-sitter-sql#236`);
- `TIMESTAMPTZ(3)` (zed#64060).
