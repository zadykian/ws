# The Mac's keyboard

The `keyboard` role sets Caps Lock, the language indicator and the symbolic hotkeys by
[`mise.macos.toml`](mise.macos.toml). mise writes each preference that is missing or differs, as the
user, with no sudo. The role has no unit of ws's. The keys read the values after a log out, as the
runbook says ([what stays by hand](../../runbook.md)).

## Caps Lock

On the built-in keyboard, `0-0-0`, Caps Lock acts as Control. macOS keeps the mapping per host, in
`NSGlobalDomain`'s `com.apple.keyboard.modifiermapping.0-0-0`, so the entry has `host = "current"`.
Its two codes, 30064771129 for Caps Lock and 30064771300 for Control, are integers.

## The language indicator

`TSMLanguageIndicatorEnabled` is the string `"0"`, which hides the indicator that macOS shows when
the input source changes. mise compares the type, so an integer 0 would read as different.

## The hotkeys

`AppleSymbolicHotKeys` of `com.apple.symbolichotkeys` holds a dictionary of hotkeys, which macOS adds
to as it changes. The role never writes it whole, as that would drop the IDs that macOS adds. Each
hotkey is an entry with a `path` into the dictionary, and mise changes that one value and keeps the
rest. A path that names an ID macOS lacks makes it.

- On, by the ID's whole value, of the type `standard`: 61, the next input source, on Command+Space,
  64, Spotlight, on Option+Space, and 160, on Command+L. mise compares the whole dictionary of an
  ID, so a value with one key more reads as different, and a run writes it again.
- Off, by `enabled` alone: 63 IDs, which the test lists. The path ends in `enabled`, so the hotkey
  keeps its `value` and turns on again with the key it had. 34 and 98 get no entry.

mise rejects two entries where one path holds the other, so no ID is on and off.

## The layout and the input sources

The layout `EngRusSimple.bundle`, with its six files, and the input sources that name it are on the
maintainer's Mac alone. They wait for the bundle's files, under #1536: the role holds none of them.
The enabled input sources hold Apple's layouts until then, as turning those off would leave no layout
of ws's to take their place.
