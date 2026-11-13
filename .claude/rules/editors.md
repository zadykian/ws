---
paths:
  - "roles/client/macos/zed/files/**"
  - "roles/server/ubuntu/zed/files/settings.json.tmpl"
  - "roles/server/ubuntu/fresh/files/config.json.tmpl"
  - "roles/server/ubuntu/lazygit/files/config.yml"
---

# editors

Zed and Fresh each have a plain file for their settings and keys: `roles/client/macos/zed/files/` and
`roles/server/ubuntu/zed/files/settings.json.tmpl` for Zed, and
`roles/server/ubuntu/fresh/files/config.json.tmpl` for Fresh. No code renders them.

A change to a shared value or an Alt or F key goes to both files. Where one file keeps a key out,
its block of keys left out, at the file's end, records why.

They share:

- the theme, the ruler and auto-save;
- the language servers' arguments;
- the Alt and F keys. The Cmd and Ctrl keys are each editor's own. Zed's stay its macOS
  defaults, but for Cmd+T, its new terminal. Fresh's stay its defaults, but for four guards in its
  terminal. No Cmd key reaches Fresh, so it has no Cmd+T.

lazygit's `roles/server/ubuntu/lazygit/files/config.yml` takes the theme's colours too, each beside
the Zed key it comes from, so a change of theme goes to it as well
([lazygit](../../roles/server/ubuntu/lazygit/README.md#the-theme)).

Fresh spells its keys as follows ([Fresh](../../roles/server/ubuntu/fresh/README.md#the-keys)):

- A stroke with Shift and a sign gets two forms: tmux sends the sign without Shift, and a direct
  terminal sends the key with Shift.
- alt+shift+space arrives as alt+space, so the map binds both.
- The shell keys are `noop` in `terminal`, so that the shell in Fresh's terminal gets them. Fresh's
  own ways out of the terminal are not among them: ctrl+space, ctrl+], F9 and alt+o.
- ctrl+h and ctrl+backspace take no binding: the default map binds both, and a map's binding of
  either warns at each start.

This rule reaches Claude alone. Nothing fails on a hand edit that leaves the two files apart.
