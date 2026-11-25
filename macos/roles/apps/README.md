# apps on the Mac

The `apps` role installs the Mac's apps, as mise's casks by `brew-cask`, which needs no Homebrew.
It pins some of them in the Dock, opens seven at login and sets Stats's settings. Its one file,
[`mise.apps.toml`](mise.apps.toml), holds no unit of ws's.

## The env

Only the env `apps` loads the file. `ws macos setup` runs the bootstrap, then `mise -E ws,apps
bootstrap` ([The Mac's frame](../frame/README.md#the-apps-env)). mise ends a run at its first
failed step, and casks come before ws's units. With the casks in the first run, a failed one
would stop the tunnel's units. Here it fails setup, with ws's units set up.

CI's `client_tree` drops the role from the tree it unpacks, so that neither Mac job downloads the
15 casks ([Checks](../../../.github/workflows/README.md#the-mac)).

## The casks

`claude`, `claude-code`, `discord`, `google-chrome`, `logi-options+`, `rectangle-pro`,
`spotify`, `stats`, `steam`, `telegram-desktop`, `transmission`, `vlc`, `whatsapp`,
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

## The Dock

`[bootstrap.settings.macos.dock] apps` pins Safari, Telegram Desktop, Discord, Claude, Zed,
Ghostty, Mail, Calendar and Yandex Telemost, in that order. mise needs each to exist before it
writes a changed layout. Zed and Ghostty come from [their roles](../zed/README.md) in the first
run, the rest from these casks or from the system. A cask's app takes its target's name:
`Telegram Desktop.app` and `Yandex Telemost.app`.
[`TestEachDockAppIsInstalledBeforeTheDockIsWritten`](apps_test.go) fails on a pinned app that no
cask installs.

mise does not restart the Dock: the layout shows after it starts again.

## The login items

`[bootstrap.settings.macos.login_items] apps` opens Claude, Discord, Mail, Rectangle Pro, Stats,
Telegram Desktop and Yandex Telemost at login. Each is an app of a cask above or the system's
Mail, and [`TestEachLoginItemIsAnInstalledApp`](apps_test.go) fails on one that neither installs.

- `exclusive = true` makes the list ws's alone. ws sets up the Mac, so an item that someone added
  by hand goes at the next run.
- mise reads the list by `LSSharedFileList` alone. An app that registers itself by `SMAppService`
  is outside the list, and `exclusive` leaves it.
- Stats opens by its item here. Its own Start at login registers it by `SMAppService`, and only a
  click in its window does that, so that box stays off.

## Stats

Stats shows the load of the CPU and the memory in the menu bar.
`[bootstrap.settings.macos.defaults."eu.exelban.Stats"]` holds 81 keys of the maintainer's export,
each of the type that the export holds: a bool, an integer or a string. Stats reads a value of
another type as missing, and mise compares the type too.
[`TestStatsKeysKeepTheExportsTypes`](apps_test.go) fails on a key of another type or one it does
not list.

- Stats reads its settings as it starts. The first run installs it and writes them, and Stats
  opens with them at the next login. A key that a later run changes applies when Stats opens again.
- `setupProcess` skips its setup window, whose first page would set each module's state and widget
  again on a click.
- Stats writes seven keys of the export itself, so they stay out, and
  [`TestStatsHoldsNoStateKey`](apps_test.go) fails on one. A run would write each back after Stats
  changed it:
  - `version`, at its first start after each update;
  - `support_ts` and `updater_check_ts`, the times of its support window and of its check for
    updates;
  - `runAtLoginInitialized`, set by a click on its Start at login. Its start then moves the login
    item to `SMAppService`, and ws sets the login item here;
  - `remote_tokens_migrated_to_keychain`, set once its account's tokens are in the keychain;
  - the `NSWindow Frame` and `NSToolbar Configuration` of its settings window, which AppKit writes,
    with the screen's size.
- `Battery_state` is true, from a MacBook. On a Mac with no battery, Stats writes it false as it
  starts, and each run writes it back.
- The export sets no `update-interval`, so Stats installs its updates itself, as the cask's
  `auto_updates` says.
