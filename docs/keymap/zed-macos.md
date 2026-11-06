# The Zed keymap on a Mac

F12's keys reach Zed through a Mac keyboard and macOS. This page lists what stops them.

## Keys a Mac keyboard lacks

Six F12 keystrokes drop from Zed's keymap:

| F12 action | F12's key | Lacks |
| --- | --- | --- |
| `Generate` | Alt+Insert | Insert |
| `NewElement` | Alt+Insert | Insert |
| `ShowPopupMenu` | Menu | the menu key |
| `Pause` | Ctrl+Alt+Break | Break |
| `Pause` | Ctrl+Pause | Pause |
| `Terminal.Paste` | Shift+Insert | Insert |

Zed does not tell numpad keys apart ([zed#11948](https://github.com/zed-industries/zed/issues/11948)).

## The F-keys and Fn

F-keys need Fn, or the setting "Use F1, F2, etc. keys as standard function keys" in System
Settings. The name of that setting is unchecked: it comes from Linux. `defaults write -g
com.apple.keyboard.fnState -bool true` should do the same, and is unchecked too.

## macOS's own shortcuts on F12's keys

Apple lists its shortcuts at <https://support.apple.com/en-us/102650>. Spaces' keys are in System
Settings, under the Mission Control shortcuts, not on that page.

| Key | macOS | F12 action |
| --- | --- | --- |
| Ctrl+Space | previous input source | Completion (Zed's own default too) |
| Shift+Ctrl+Space | input source, by Zed's comment | Parameter info |
| Ctrl+Alt+Space | next input source | Type-matching completion |
| Ctrl+Up, Ctrl+Down | Mission Control, app windows | Scroll line, overloads |
| Ctrl+Left, Ctrl+Right | move a space (on by default; unchecked) | Word left, right |
| Ctrl+F3, Ctrl+F4, Ctrl+F5 | focus Dock, window, toolbar | Find word, Close tab, Debug |

They go off by hand, in System Settings.

`defaults write com.apple.symbolichotkeys` can do it. The ids are from memory and unchecked:

- 60 and 61 are the input sources.
- 32 and 33 are Mission Control.
- 79 and 81 are Spaces.
- 8, 9 and 10 are the focus keys.

A change needs a re-login, or `activateSettings -u`. That is why ws does not do it.
