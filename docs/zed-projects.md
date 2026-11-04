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

## An Ansible project

The project's own virtualenv holds `ansible-core` and `ansible-lint`: ws installs none.

```jsonc
// Zed's settings for an Ansible project, which Zed on the Mac and its server on ws both read.
{
  "file_types": {
    // Zed matches a glob against the path from the worktree's own name, so each path glob starts with **/.
    "Ansible": [
      "site.yml",
      "**/group_vars/*.yml",
      "**/roles/*/defaults/*.yml",
      "**/roles/*/handlers/*.yml",
      "**/roles/*/meta/*.yml",
      "**/roles/*/tasks/*.yml",
      "**/roles/*/vars/*.yml"
    ],
    // Zed's YAML language has no suffix for these two files.
    "YAML": [".ansible-lint", ".yamllint"]
  },
  "lsp": {
    // The Ansible extension's key is "ansible", not the "ansible-language-server" of Zed's docs.
    "ansible": {
      // The project's collections, at its requirements' pins. ALS runs ansible from the
      // workspace folder, against which ansible resolves this relative path.
      "binary": { "env": { "ANSIBLE_COLLECTIONS_PATH": "collections" } },
      "settings": {
        // The project's venv, with ansible-core and ansible-lint. ALS puts its bin first on PATH,
        // so "ansible" and "ansible-lint" resolve there.
        "python": { "interpreterPath": "${workspaceFolder}/.venv/bin/python" }
      }
    }
  },
  // A manual format of a YAML file runs Prettier, which breaks {{ }} (zed#46632).
  "languages": { "YAML": { "prettier": { "allowed": false } } }
}
```

A project adapts three parts of the file. `site.yml` becomes its playbooks' names, with a glob for
each directory of Ansible files it has, such as `host_vars`. The two paths become its own
collections and venv. A role that adds a directory needs a glob, or its files stay YAML.

Why each key:

- **`**/` first:** Zed matches a glob against the file name and the path from the worktree's name,
  so `roles/*/tasks/*.yml` never matches. A user glob wins a tie with a built-in suffix.
- **No `*.j2`:** ALS would lint the templates. Do not install the `jinja2` extension beside
  `ansible`, as both define "Jinja2".
- **`lsp.ansible`:** the extension reads that key. The docs' `lsp."ansible-language-server"` is
  silently ignored. `binary.env` merges with the `path` and `arguments` of ws's file.
- **The collections path:** without it, ansible-lint passes but cannot load a collection's plugin,
  and nothing shows.
- **`${workspaceFolder}`:** Zed replaces it in `python.interpreterPath` alone, and ALS runs every
  command from the workspace folder.
- **`ansible-lint`:** ALS runs it with the `.ansible-lint` found up to the workspace root, offline.
  The project's configuration governs, and `lint.arguments` stays empty.
- **`prettier.allowed` false:** a manual format of a YAML file runs Prettier by default, and it
  breaks `{{ }}` (zed#46632). `format_on_save` is off by default.
- **Layers:** if the Mac's `settings.json` sets `file_types.YAML`, this file's array wins.

Check the file:

- It parses once its full-line comments are off, with no duplicate key. Copy the block to
  `settings.json`, then run:

  ```sh
  grep -v '^ *//' settings.json | jq -e . >/dev/null
  grep -v '^ *//' settings.json | python3 -I -c 'import json,sys; json.load(sys.stdin, object_pairs_hook=lambda p: len({k for k, _ in p}) < len(p) and sys.exit("duplicate key") or dict(p))'
  ```

- In Zed, `site.yml` and `roles/NAME/tasks/main.yml` say Ansible in the status bar. A workflow
  under `.github/workflows` and `.ansible-lint` say YAML.
- `dev: open language server logs` lists `ansible`, and `pgrep -af ansible-language-server` on ws
  shows the binary under `~/.local/lib/zed-servers`.

Limits:

- ALS drops its lint results after an edit (zed#55730). Reopen the file or restart the server.
  Running ansible-lint on the file is the check.
- A restored buffer may open as YAML (zed#59487): reopen it. yaml-language-server indents by 3
  (zed#54043).
- Make has no server. Its recipe lines get run buttons that fail (zed#12688,
  `caius/zed-make#30`): use the buttons on target lines.
- The paths `collections` and `.venv`, and ALS 26.6.0 end to end in Zed, are untested.
