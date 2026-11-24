# apps on the Mac

The `apps` role installs the Mac's apps, as mise's casks by `brew-cask`, which needs no Homebrew.
Its one file, [`mise.apps.toml`](mise.apps.toml), holds no unit of ws's.

## The env

Only the env `apps` loads the file. `ws macos setup` runs the bootstrap, then `mise -E ws,apps
bootstrap` ([The Mac's frame](../frame/README.md#the-apps-env)). mise ends a run at its first
failed step, and casks come before ws's units. With the casks in the first run, a failed one
would stop the tunnel's units. Here it fails setup, with ws's units set up.

CI's `client_tree` drops the role from the tree it unpacks, so that neither Mac job downloads the
14 casks ([Checks](../../../.github/workflows/README.md#the-mac)).

## The casks

`claude`, `claude-code`, `discord`, `google-chrome`, `logi-options+`, `rectangle-pro`,
`spotify`, `steam`, `telegram-desktop`, `transmission`, `vlc`, `whatsapp`,
`winbox` and `yandextelemost`. Each takes `latest`: a cask exists at its current version alone,
and `mise bootstrap packages upgrade` upgrades them.

- mise checks each download against the SHA-256 of its cask, but for four that list none:
  `google-chrome`, `logi-options+`, `spotify` and `yandextelemost`. There it
  checks no signature, and the file it downloads carries no quarantine. So these four are outside
  the rule of the pinned downloads, as is each cask at its latest
  ([CLAUDE.md](../../../.claude/CLAUDE.md)).
- `claude-code` is a cask, not a role: it has a SHA-256 and links `claude` into
  `/opt/homebrew/bin`, which devtools' `/etc/paths.d/ws` puts on the `PATH`. The server's
  installer has none.
- Logi's installer calls sudo on a first run, and asks for a reboot. A run with no terminal needs a
  sudo that asks no password.
- Not send-to-kindle: its download is refused (403), and mise ends a run at the first failed cask.
- Not qBittorrent: Homebrew disabled its cask on 2026-09-01, as it fails Gatekeeper.
- No firewall exception: replies pass block-all
  ([the firewall](../firewall/README.md#what-keeps-working)).
