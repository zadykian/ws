---
paths:
  - "roles/client/macos/mac/zed/**"
  - "roles/server/ubuntu/zed/files/settings.json.tmpl"
  - "roles/server/ubuntu/fresh/files/config.json.tmpl"
  - "roles/server/ubuntu/lazygit/files/config.yml"
---

# editors

Zed and Fresh each have a plain file for their settings and keys: `roles/client/macos/mac/zed/` and
`roles/server/ubuntu/zed/files/settings.json.tmpl` for Zed, and
`roles/server/ubuntu/fresh/files/config.json.tmpl` for Fresh. No code renders them.

A change to a shared value or a key goes to both files. Where one file keeps a key out, its block
of keys left out, at the file's end, records why.

They share:

- the theme, the ruler and auto-save;
- the language servers' arguments;
- the keys.

lazygit's `roles/server/ubuntu/lazygit/files/config.yml` takes the theme's colours too, each beside
the Zed key it comes from, so a change of theme goes to it as well
([lazygit](../../docs/lazygit.md#the-theme)).

Fresh spells its keys as follows ([Fresh](../../docs/fresh.md#the-keys)):

- A stroke with Shift and a sign gets two forms: tmux sends the sign without Shift, and a direct
  terminal sends the key with Shift.
- Some chords arrive as another key, so each is bound under both:
  - ctrl+alt+4 as ctrl+alt+\\, and ctrl+alt+/ as ctrl+alt+shift+-;
  - ctrl+k ctrl+/ as ctrl+k ctrl+shift+-;
  - ctrl+shift+space as ctrl+space, and alt+shift+space as alt+space.
- The shell keys are `noop` in `terminal`, so that the shell in Fresh's terminal gets them. Fresh's
  own ways out of the terminal are not among them: ctrl+space, ctrl+], F9 and alt+o.
- ctrl+h is bound outside the map, as the default map binds it with ctrl+backspace and warns at
  each start. ctrl+backspace and ctrl+shift+backspace take no binding.

This rule reaches Claude alone. Nothing fails on a hand edit that leaves the two files apart.
