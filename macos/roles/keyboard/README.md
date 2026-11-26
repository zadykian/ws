# The Mac's keyboard

The `keyboard` role sets Caps Lock, the language indicator, the symbolic hotkeys and the input
sources by [`mise.macos.toml`](mise.macos.toml). mise writes each preference that is missing or
differs, as the user, with no sudo. ws's units write the layout that the input sources name.
The keys read the values after a log out, as the runbook says ([what stays by hand](../../runbook.md)).

## Caps Lock

On the built-in keyboard, `0-0-0`, Caps Lock acts as Control. macOS keeps the mapping per host, in
`NSGlobalDomain`'s `com.apple.keyboard.modifiermapping.0-0-0`, so the entry has `host = "current"`.
Its two codes, 30064771129 for Caps Lock and 30064771300 for Control, are integers.

The mapping is the built-in keyboard's alone: macOS keys it by a keyboard's vendor and product. So
another keyboard, such as one that Logi Options+ pairs, keeps its Caps Lock until the role gets an
entry for its key. That `0-0-0` names the built-in keyboard is for trial 4 to show on the Mac.

## The language indicator

`TSMLanguageIndicatorEnabled` is the string `"0"`, which hides the indicator that macOS shows when
the input source changes. mise compares the type, so an integer 0 would read as different.

## The hotkeys

`AppleSymbolicHotKeys` of `com.apple.symbolichotkeys` holds a dictionary of hotkeys, which macOS adds
to as it changes. The role never writes it whole, as that would drop the IDs that macOS adds. Each
hotkey is an entry with a `path` into the dictionary, and mise changes that one value and keeps the
rest. A path that names an ID macOS lacks makes it.

- On, by the ID's whole value, of the type `standard`: 61, the next input source, on Command+Space,
  64, Spotlight, on Option+Space, and 160, Launchpad, on Command+L. mise compares the whole
  dictionary of an ID, so a value with one key more reads as different, and a run writes it again.
- Off, by `enabled` alone: 63 IDs, which the test lists. The path ends in `enabled`, so the hotkey
  keeps its `value` and turns on again with the key it had.
- As macOS has them, by no entry: 34 and 98. The maintainer keeps both on at macOS's own value and
  key, which an entry would replace with ws's.

mise rejects two entries where one path holds the other, so no ID is on and off.

Launchpad is Apps since macOS 26. A hotkey of the system comes before an app's key, so Command+L
no longer opens Safari's location or selects Zed's line. Only a Mac shows what Command+L opens on
macOS 27, and what Safari and Zed lose: trial 4 of [the checks](../../checks/README.md#the-trials)
notes both.

## The layout

`/Library/Keyboard Layouts/ws.bundle` is the maintainer's English and Russian layout, which
Ukelele first wrote. Its files are in [`files/`](files/ws.bundle): `Info.plist`, `version.plist`,
`ENG.keylayout`, `RUS.keylayout` and their icons. Each is a `file` unit of
[`units.toml.tmpl`](units.toml.tmpl), root's, readable by all and copied byte for byte, with no
mark of ws's, as a layout takes none.
The test pins the units and each file's bytes. The bundle holds no `InfoPlist.strings`: Ukelele's
two named each layout by its own name, which changed nothing.

The bundle is `ws` 1.0.0. Its path, its `CFBundleName` and its `ProjectName` say `ws`. Its
`CFBundleVersion`, `BuildVersion` and `SourceVersion` say 1.0.0. Its `CFBundleIdentifier` and the
two layouts' `TISInputSourceID` keep Ukelele's: macOS names an input source in use by its ID, and
trial 3 reads it so.

Each keylayout holds what its keys reach, and a test keeps it so. `ENG.keylayout` lost Ukelele's
five dead keys, which no key reached, their accents in the actions, their terminators and the JIS
key map set, which no layout named. So no key types two units, and its `maxout` is 1.

Each key map types F1 to F20 as `&#x0010;`, the code of a function key, and a test keeps it so.
Ukelele's files left F17 to F20 out of most maps, where those keys typed no character. No key of a
map types nothing: RUS's map for Command, Option and Control types ENG's `§` and the keypad's `.`.

## The icons

`ENG.icns` and `RUS.icns` are the layouts' icons. macOS finds each by its keylayout's name, in
`Resources`, as Apple's TN2056 says. Each draws a square with round corners and cuts its letters
out of it: `A` for ENG and `РУ` for RUS, as Apple's ABC and Russian icons do. Each is a template, as
`TISIconIsTemplate` says: black with alpha, which macOS tints to fit the menu bar.

A Go program drew them once, from DejaVu Sans Bold 2.37, and ws keeps the icons alone. The square
fills the side, and its corners' radius is 7/32 of it. The letters stand in its middle, their cap
height half the side. `РУ` is narrowed to 0.79 of its width, so that its ink spans 3/4 of the side.
Each file holds `ic04` and `ic05`, 16 and 32 pixels in ARGB, and `ic11` and `ic12`, 32 and 64
pixels in PNG, for a Retina screen. A test checks each entry, its side and its black.

## The input sources

`com.apple.inputsources` names the layout's two sources in `AppleEnabledThirdPartyInputSources`:
`ENG`, -20611, and `RUS`, -21268, each an `InputSourceKind` of `Keyboard Layout`. The IDs and names
are those of the two keylayout files. macOS reads a third party's layout there from macOS 12 on.
`com.apple.HIToolbox`'s `AppleEnabledInputSources` holds the same two first, as macOS lists a
layout before a method. The character palette and press-and-hold follow, as neither types a
letter. So `ENG` and `RUS` are the only layouts on, and no layout of Apple's stays.

The role sets no key of the source in use or of the last ones used. macOS rewrites each at a
switch, and mise would find it changed at the next run.

mise writes each array whole, and compares it element by element, in order, and by type: an ID is
an integer, and a key order is no difference. It writes the preferences before ws's units, so
that the sources name the layout for a moment before its files exist, and the log in reads both.

A layout of Apple's would come back from the login window's sources. The login window and
FileVault's unlock type with the system's, such as Setup Assistant's ABC, which
`/Library/Preferences/com.apple.HIToolbox.plist` holds. Neither ws nor mise writes that file: mise
writes the user's preferences alone. If a log in carries ABC into the user's array, the array
differs, and setup's plan says so after each log in.
[Trial 3](../../checks/README.md#the-trials) shows whether macOS does so.
