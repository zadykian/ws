# Keys a terminal or tmux may not pass on

A terminal that reports modified keys as `CSI u`, or answers helix's request for the kitty
keyboard protocol, sends every key of the keymap to a helix running in it directly. One that sends
keys as older terminals did loses or changes those below. So does tmux, whatever the terminal, but
for those that its `extended-keys always` and `extended-keys-format csi-u` options let through:
[helix's keys](../helix.md#helixs-keys) has them. Every other key arrived from each kind of
terminal tried, directly and through tmux. Where F12's key doesn't arrive, helix's own key does
the same: [the actions helix has](actions.md) gives it.

Each was tried with hx 25.07.1 and tmux 3.6a from Ubuntu's archive, sent as each kind of
terminal sends it. "tmux" is tmux's defaults or cld's servers.

| F12's keys | F12 action | Older terminal | tmux | tmux with the options |
| --- | --- | --- | --- | --- |
| Shift+Ctrl+T | Go to file | as Ctrl+T, Refactor this | the same | the same |
| Shift+Ctrl+F | Find in files | as Ctrl+F, Find | the same | the same |
| Shift+Ctrl+W | Shrink selection | as Ctrl+W, Extend selection | the same | the same |
| Shift+Ctrl+Z | Redo | as Ctrl+Z, Undo | the same | the same |
| Shift+Ctrl+S | Save all | as Ctrl+S, Save | the same | the same |
| Shift+Ctrl+I | `Implement methods` | as Tab | the same | the same |
| Shift+Ctrl+P | Pause program | as Ctrl+P | the same | the same |
| Shift+Ctrl+Alt with a letter | Go to symbol, and others | no | no | no |
| Ctrl+-; Ctrl+= | Back, Forward | no | no | yes |
| Ctrl+, | Search everywhere | no | no | yes |
| Ctrl+Enter; Shift+Enter | Show intention actions | as Enter | the same | yes |
| Shift+Ctrl+Enter | Start new line | as Enter | the same | yes |
| Ctrl+Alt+Enter | Reformat code | as Alt+Enter, Show intention actions | the same | yes |
| Shift+Ctrl+/ | Comment with block comment | as Backspace | the same | yes |
| Shift+Ctrl+Backspace | Last edit location | as Ctrl+H | as Ctrl+H, or not at all | yes |
| Ctrl+Alt+/; Ctrl+K, Ctrl+/ | Comment line | no | no | no |
| Shift+Ctrl+\[ | Select containing declaration | no | no | no |
| Shift+Ctrl+Alt+\[; Shift+Ctrl+Alt+\] | Code block start, end with selection | no | no | no |
| Ctrl+\[ | Containing declaration | as Esc | the same | the same |
| Ctrl+\] | Matching brace | as Ctrl+5 | the same | the same |
| Ctrl+I; Ctrl+K, Ctrl+I | Toggle inlay hints, Quick documentation | as Tab | the same | the same |
| Ctrl+M, C; Ctrl+M, Ctrl+C | Scroll to center | as Enter, then C or Ctrl+C | the same | the same |
| Shift+Ctrl+Tab | Switcher | as Shift+Tab | the same | the same |
| Ctrl+Alt+2 | Errors in solution | as Ctrl+Alt+Space, a completion key | the same | the same |
| Ctrl+Pause | Pause program | no | no | no |
| Ctrl+B | Toggle line breakpoint | yes | pressed twice, as tmux's own prefix | the same |
| Ctrl+Q | Go to action | yes | pressed twice in cld's sessions, its prefix there | yes |

helix drops Shift from a character key, space too, whatever the terminal. So Shift+Alt+Space,
Second basic completion, arrives as Alt+Space, which the old `config.toml` bound to it as well.
Shift+Ctrl+Space, Parameter info, arrives as Ctrl+Space, Completion. On a Mac, the Alt keys need
Option to act as Meta or send Esc+.
