# The Mac's desktop

The `desktop` role sets the Dock, Finder, windows, sounds and screenshots by
[`mise.macos.toml`](mise.macos.toml). mise writes each preference that is missing or differs,
as the user, with no sudo. The role's one unit of ws's sets the wallpaper
([The wallpaper](#the-wallpaper)).

## The values

- The Dock: icons at 74, no recent apps, and a window minimised into its app's icon. The bottom
  right hot corner does nothing (`wvous-br-corner` is 1).
- Finder: the column view, new windows at the home folder (`PfHm`), no recent tags.
- Windows: no tiling by a drag to an edge, and no margins between tiles. A click on the wallpaper
  shows no desktop, and widgets stay hidden on it and in Stage Manager.
- Sounds: a beep at volume 0, Tink as the beep, no sound of the interface.
- Screenshots: a selection, copied to the clipboard, and a recording to a file.
- The rest: dark mode, every extension and scroll bar shown, no swipe between pages, a trackpad
  speed of 0.5, a glass tint of 1.0 and no Siri. The screen saver starts after two hours.

## The types

mise compares a value's type too, so each value keeps the type that the Mac holds. TOML's `1.0` is
a real and its `0` an integer: `NSGlassTintAmount` and the beep's volume are reals,
`com.apple.sound.uiaudio.enabled` an integer. The test of this role pins them. A value of another
type reads as different. Then `ws macos setup --plan` exits 2
([what the Mac reports](../../README.md#what-the-mac-reports)).

## Dark mode

`AppleInterfaceStyle = "Dark"` holds while Appearance is not Auto. Where Appearance is Auto, macOS
drops the key by day, and the next run sets it again. mise deletes no key, so
`AppleInterfaceStyleSwitchesAutomatically` is set to `false`.

## The screen saver

`idleTime` of `com.apple.screensaver` is a preference of the current host, which mise writes by an
entry with `host = "current"`.

## The wallpaper

mise pours the formula `wallpaper`, 2.3.4 at this writing, with its `brew` manager, which needs no
Homebrew. The formula takes `latest`, as Homebrew keeps the current bottle alone, and mise checks
it against the SHA-256 of that bottle ([devtools](../devtools/README.md#the-formulae)).
`mise bootstrap packages upgrade` upgrades it.

[`state/wallpaper`](units.toml.tmpl) runs `wallpaper set` with Stone, the system's
`/System/Library/Desktop Pictures/Solid Colors/Stone.png`, as the user. A unit's `PATH` lacks
`/opt/homebrew/bin`, so the unit's `path` names it. The check runs `wallpaper get`, which prints a
line for each screen, and wants one line at least and each line Stone's. So a first run sets Stone
on every screen, and a run with Stone there changes nothing and asks no password.

Both cover each screen's current Space alone, the one of a clean Mac.

CI's `client_tree` drops the unit and the formula from the tree it unpacks: a runner has no screen,
so `wallpaper get` prints no line and the check fails each run.

## The restart

mise restarts neither the Dock nor Finder. They read their preferences again at the next log out, so
the first run's values show after one.
