# The Mac's power and updates

`mise bootstrap` keeps the display on while the Mac is on power, and stops the App Store's updates,
by two units of the role `power`. Each sets one fact, as a `state` unit that runs a command
([The state kind](../../../engine/core/kind/state.md)). The check runs as the user, and the
command as root through sudo, so a run that changes nothing asks no password.

## state/power-display

```sh
pmset -c displaysleep 0
```

`-c` is the mains' setting alone, so the battery keeps its own sleep. The check reads
`pmset -g custom`, which lists each source's settings under its heading, and wants `displaysleep 0`
under `AC Power:`. The command runs where that line is missing or reads another number. A Mac on
power then never turns its display off, as a session that the server's work keeps open needs.

## state/power-store

```sh
defaults write /Library/Preferences/com.apple.commerce AutoUpdate -bool false
```

The key stops the App Store's automatic updates of apps. `defaults read` prints a false boolean
as `0`, and fails where the key is missing, which the check reads as on. The tools that mise
installs are not the App Store's: [the runbook](../../runbook.md#what-stays-by-hand) gives
their upgrade.

## macOS 27

Apple's documents mark `AutomaticallyInstallAppUpdates` as removed from macOS 27, which may read
another key than `AutoUpdate`. The unit writes `AutoUpdate`, and the real Mac shows whether that
key works.

## CI

The runner's VM shows `displaysleep 0` under `AC Power:` before the first run, and holds
`AutoUpdate = 1`, so the first run writes the second alone
([the case](../../../.github/client/88-power.sh)).
