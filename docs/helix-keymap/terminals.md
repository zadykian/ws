# Keys a terminal or tmux may not pass on

A terminal that reports modified keys as `CSI u`, or answers helix's request for the kitty
keyboard protocol, sends every key of the keymap to a helix running in it directly. One that sends
keys as older terminals did loses or changes those below. So does tmux, whatever the terminal, but
for those that its `extended-keys always` and `extended-keys-format csi-u` options let through:
[helix's keys](../helix.md#helixs-keys) has them. Every other key arrived from each kind of
terminal tried, directly and through tmux; the JetBrains terminal, below, loses more. Where F12's
key doesn't arrive, helix's own key does the same: the tables below and
[the actions helix has](actions.md) give it.

Each was tried with hx 25.07.1 and tmux 3.6a from Ubuntu's archive, sent as each kind of
terminal sends it. "tmux" is tmux's defaults or cld's servers.

<!-- Implement methods is an IDE action's name, which write-good takes for a wordy verb. -->
<!-- vale write-good.TooWordy = NO -->

| F12's keys | F12 action | Older terminal | tmux | tmux with the options |
| --- | --- | --- | --- | --- |
| Shift+Ctrl+T | Go to file | as Ctrl+T, Refactor this | the same | the same |
| Shift+Ctrl+F | Find in files | as Ctrl+F, Find | the same | the same |
| Shift+Ctrl+W | Shrink selection | as Ctrl+W, Extend selection | the same | the same |
| Shift+Ctrl+Z | Redo | as Ctrl+Z, Undo | the same | the same |
| Shift+Ctrl+S | Save all | as Ctrl+S, Save | the same | the same |
| Shift+Ctrl+I | Implement methods | as Tab | the same | the same |
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

<!-- vale write-good.TooWordy = YES -->

helix drops Shift from a character key, space too, whatever the terminal. So Shift+Alt+Space,
Second basic completion, arrives as Alt+Space, which `config.toml` binds to it as well.
Shift+Ctrl+Space, Parameter info, arrives as Ctrl+Space, Completion. On a Mac, the Alt keys need
Option to act as Meta or send Esc+.

## The JetBrains terminal

The terminal of GoLand and Rider 2026.2 sends keys as older terminals do, with tmux's options or
without, so it loses the keys above as they do. By JediTerm's source, read but not run, it loses
more:

- with Ctrl, it drops Alt: Ctrl+Alt or Shift+Ctrl+Alt with a letter arrives as Ctrl with the
  letter;
- with Shift, it drops Alt too: Shift+Alt with a character arrives as the character, which
  insert mode types;
- Alt with an arrow, Home, End, PgUp, PgDn, Insert, Delete or an F key arrives as Esc and the key
  with no modifier. Neither helix nor tmux reads that as one key: helix leaves insert mode, as for
  Esc, and does nothing else. Alt+Left and Alt+Right alone arrive, but as Alt+B and Alt+F on a Mac.

tmux passes each on as it arrives. The IDE may also take a key of its own keymap before the
terminal does, unless the terminal's Override IDE shortcuts lets it through. These are the keys of
the keymap that the terminal changes, with what each runs then, in normal mode for Shift+Alt, and
helix's own key for the action:

| F12's keys | F12 action | Arrives as | Which runs | helix's own key |
| --- | --- | --- | --- | --- |
| Ctrl+Alt+D | Go to implementation | Ctrl+D | `page_cursor_half_down`; in insert mode, `delete_char_forward` | `g i` |
| Ctrl+Alt+C | Introduce constant | Ctrl+C | Copy; nothing in insert mode | `Space a` |
| Ctrl+Alt+V | Inspection results | Ctrl+V | Paste | `Space D` |
| Ctrl+Alt+Z | Rollback lines | Ctrl+Z | Undo | `:reset-diff-change` |
| Ctrl+Alt+B | Edit breakpoint | Ctrl+B | Toggle line breakpoint | `Space G`, then Ctrl+C |
| Shift+Ctrl+Alt+T | Go to symbol | Ctrl+T | Refactor this | `Space S` |
| Shift+Ctrl+Alt+F | Go fmt file | Ctrl+F | Find | `:format` |
| Shift+Ctrl+Alt+L | Reformat file | Ctrl+L | nothing; insert mode types `l` | `:format` |
| Shift+Ctrl+Alt+N; Shift+Ctrl+Alt+P | Next change, Previous change | Ctrl+N, Ctrl+P | nothing; insert mode types the letter | `]g`, `[g` |
| Shift+Ctrl+Alt+R | Split right | Ctrl+R | the start of Rename's chord | `Space w v` |
| Shift+Alt+I | Inspect code with editor settings | `I` | `insert_at_line_start` | `Space d` |
| Shift+Alt+L | Locate in solution view | `L` | nothing | `Space E` |
| Shift+Alt+. | Add selection for next occurrence | `>` | `indent` | `*`, then `v n` |
| Shift+Alt+, | Unselect occurrence | `<` | `unindent` | Alt+, |
| Shift+Alt+; | Select all occurrences | `:` | the command line | `Space h` |
| Shift+Alt+= | Extend selection | `+` | nothing | Alt+O |
| Shift+Alt+- | Shrink selection | `_` | `trim_selections` | Alt+I |
| Shift+Alt+/ | Find next | `?` | `rsearch` | `n` |
| Shift+Alt+\[ | Code block start | `{` | nothing | Alt+B |
| Shift+Alt+\] | Select containing declaration | `}` | nothing | `m a f` |
| Shift+Alt+Space | Second basic completion | Space | the space menu | Ctrl+Space |
| Alt+Up; Alt+Down | Move line up, down | no | | `X d k P`, `X d p` |
| Alt+PgUp; Alt+PgDn | Previous error, Next error | no | | `[d`, `]d` |
| Alt+F12 | Quick definition | no | | `g i` |
| Shift+Alt+F12; Shift+Ctrl+Alt+F12 | Show usages, Find usages settings | no | | `g r` |
| Alt+F11 | Inspect code | no | | `Space D` |
| Shift+Alt+F2; Shift+Alt+PgUp; Shift+Alt+PgDn | Previous error, Next error in solution | no | | `Space D` |
| Alt+Insert | Generate | no | | `Space a` |
| Alt+F5 | Stop | no | | `Space G t` |
| Alt+F9 | Edit breakpoint | no | | `Space G`, then Ctrl+C |
| Shift+Alt+Up; Shift+Alt+Down | Clone caret above, below | no | | Alt+C, `C` |
| Shift+Alt+Left; Shift+Alt+Right | Previous splitter, Next splitter | no | | Ctrl+Home, Ctrl+End: F12's other keys |
| Ctrl+Alt+PgUp; Ctrl+Alt+PgDn | Previous tab, Next tab | no | | `g p`, `g n` |
| Alt+Left; Alt+Right | Previous tab, Next tab | on a Mac, Alt+B and Alt+F | Code block start, and nothing | `g p`, `g n` |
