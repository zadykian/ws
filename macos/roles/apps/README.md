# apps on the Mac

The `apps` role installs the Mac's apps, as mise's casks by `brew-cask`, which needs no Homebrew.
It pins some of them in the Dock, opens seven at login and sets Stats's settings. Its one file,
[`mise.apps.toml`](mise.apps.toml), holds no unit of ws's.

## The env

Only the env `apps` loads the file. `ws macos setup` runs the bootstrap, then `mise -E ws,apps
bootstrap`. A failed cask fails setup, with ws's units set up
([The Mac's frame](../frame/README.md#the-apps-env)).

CI's `client_tree` drops the role from the tree it unpacks, so that neither Mac job downloads the
15 casks ([Checks](../../../.github/workflows/README.md#the-mac)).

## The casks

`claude`, `claude-code`, `discord`, `google-chrome`, `logi-options+`, `rectangle-pro`,
`spotify`, `stats`, `steam`, `telegram-desktop`, `transmission`, `vlc`, `whatsapp`,
`winbox` and `yandextelemost`. Each takes `latest`: a cask exists at its current version alone.
[The runbook](../../runbook.md#what-stays-by-hand) gives the upgrade. A cask that updates itself
is left to itself.

- mise strips the quarantine from every cask's download and app, and checks no signature on any,
  so Gatekeeper assesses none. It checks eleven downloads against the SHA-256 that Homebrew's API
  gives at run time, an unsigned document. Four list none: `google-chrome`, `logi-options+`,
  `spotify` and `yandextelemost`. All fifteen are outside the rule of the pinned downloads
  ([CLAUDE.md](../../../.claude/CLAUDE.md)).
- `claude-code` is a cask, not a role: it has a SHA-256 and links `claude` into
  `/opt/homebrew/bin`, which devtools' `/etc/paths.d/ws` puts on the `PATH`. The server's
  installer has none.
- Logi Options+ stays a cask. Its installer is a download with no SHA-256 and no quarantine,
  which mise runs through sudo on a first run, and it asks for a reboot. A run with no terminal
  needs a sudo that asks no password.
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
  is outside the list, and `exclusive` leaves it. An app that adds its own item to the list at
  each start loses it at each run, and the plan exits 2 until then. Trial 5 records which casks
  do it ([the trials](../../checks/README.md)).
- Stats opens by its item here. Its own Start at login registers it by `SMAppService`, and only a
  click in its window does that, so that box stays off.

## Stats

Stats shows the load of the CPU and the memory in the menu bar.
`[bootstrap.settings.macos.defaults."eu.exelban.Stats"]` holds 116 keys of the maintainer's
export, each of the type that the export holds: a bool, an integer, a string, or an array for
`CombinedModules_modifierKeys`. Stats reads a value of another type as missing, and mise compares
the type too.
[`TestStatsKeysKeepTheExportsTypes`](apps_test.go) fails on a key of another type or one it does
not list.

- Stats reads its settings as it starts. The first run installs it and writes them, and Stats
  opens with them at the next login. A key that a later run changes applies when Stats opens again.
- `setupProcess` skips its setup window, whose first page would set each module's state and widget
  again on a click.
- `CombinedModules` shows the modules in one item of the menu bar, the CPU, the memory and the
  network's speed. Its `CombinedModules_modifierKeys` is an empty array, where Stats's default is
  Option. mise writes an empty array and finds it equal to one of no items, so a second run changes
  nothing.
- Each `NAME_position` key orders a module or a widget, as a drag in Stats's window does.
- The seven `sensor_` keys pick the sensors that the Sensors module shows, which is off. They hold
  on the maintainer's Mac alone: `TH0x`, `TaLP` and `PPBR` are codes of its hardware's sensors. On
  another Mac, Stats reads no key of a sensor it lacks.
- Stats or AppKit writes eight keys of the export, so they stay out, and
  [`TestStatsHoldsNoStateKey`](apps_test.go) fails on one. A run would write each back after Stats
  changed it:
  - `version`, at its first start after each update;
  - `support_ts` and `updater_check_ts`, the times of its support window and of its check for
    updates;
  - `runAtLoginInitialized`, set by a click on its Start at login. Its start then moves the login
    item to `SMAppService`, and ws sets the login item here;
  - `remote_tokens_migrated_to_keychain`, set once its account's tokens are in the keychain;
  - the `NSWindow Frame` and `NSToolbar Configuration` of its settings window, which AppKit writes,
    with the screen's size;
  - `NSOSPLastRootDirectory`, the last folder of its Open and Save panels: a bookmark that holds the
    Mac's account and a volume's ID, as data, which mise does not write.
- `Battery_state` is false: the battery module is off. The other `Battery_` keys stay, as the
  export holds them, and apply if the module is turned on.
- The export sets no `update-interval`, so Stats installs its updates itself, as the cask's
  `auto_updates` says.
