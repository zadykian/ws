# helix's F12 keymap

F12 is the keymap GoLand and Rider use here: `F12.xml`, 736 actions over IntelliJ's default
keymap, in Visual Studio's style. The `helix` role's `config.toml` binds the keys of each F12
action that helix has a command for, in normal and insert mode. This page lists those 109
actions by topic, with helix's own key for each, then the 414 that helix has no command for.
107 of the 109 are bound: Parameter info and Open aren't, and the table says why. The
other 213 actions of `F12.xml` have no keys: they take the default keymap's keys away. An
action that `F12.xml` doesn't name keeps the default keymap's keys, which aren't here.

The keys are F12's as the IDEs write them: a comma between the keys of a chord, a semicolon
between an action's keys. F12's Cmd keys are left out, as Cmd never reaches a program in a
terminal; each has a Ctrl twin, but for those of Cut and Redo in the editor. So are its numpad
keys in the tables of actions helix has, as helix has no names for them.

## In insert mode

A key does in insert mode what it does in normal mode, but:

- the completion keys work in insert mode alone, as helix drops completions outside it;
- the keys that select leave insert mode first, since text typed in insert mode goes before the
  selection rather than over it;
- Copy and Cut do nothing: insert mode's selection is the character after the cursor, which Cut
  would take, where the IDEs copy or cut the line when nothing is selected;
- Surround with acts on that character, and Delete stays helix's, which deletes it;
- Paste inserts at the cursor through `insert_register`, rather than before the selection;
- Move line goes back to insert mode.

In insert mode, a Ctrl or Alt key with a character that nothing is bound to types that character,
as helix does with any such key. So does a chord's second key that the chord lacks, after the
first key's character.

## Keys a terminal or tmux may not pass on

A terminal that reports modified keys as `CSI u`, or answers helix's request for the kitty
keyboard protocol, sends every key on this page to a helix running in it directly. One that sends
keys as older terminals did loses or changes those below, and so does tmux, whatever the terminal,
but for those that its `extended-keys always` and `extended-keys-format csi-u` options let
through: the README has them. Every other key arrived from each kind of terminal tried, directly
and through tmux; the JetBrains terminal, below, loses more. Where F12's key doesn't arrive,
helix's own key, in the tables below, does the same.

Each was tried with hx 25.07.1 and tmux 3.6a from Ubuntu's archive, sent as each kind of
terminal sends it. "tmux" is tmux's defaults or cld's servers.

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

helix drops Shift from a character key, space too, whatever the terminal: Shift+Alt+Space, Second
basic completion, arrives as Alt+Space, to which it is bound, and Shift+Ctrl+Space, Parameter
info, as Ctrl+Space, Completion. On a Mac, the Alt keys need Option to act as Meta or send Esc+.

### The JetBrains terminal

The terminal of GoLand and Rider 2026.2 sends keys as older terminals do, with tmux's options or
without, so it loses the keys above as they do. By JediTerm's source, read but not run, it loses
more:

- with Ctrl, it drops Alt: Ctrl+Alt or Shift+Ctrl+Alt with a letter arrives as Ctrl with the
  letter;
- with Shift, it drops Alt too: Shift+Alt with a character arrives as the character, which
  insert mode types;
- Alt with an arrow, Home, End, PgUp, PgDn, Insert, Delete or an F key arrives as Esc and the key
  with no modifier, which neither helix nor tmux reads as one key: helix leaves insert mode, as for
  Esc, and does nothing else. Alt+Left and Alt+Right alone arrive, but as Alt+B and Alt+F on a Mac.

tmux passes each on as it arrives. The IDE may also take a key of its own keymap before the
terminal does, unless the terminal's Override IDE shortcuts lets it through. These are the keys of
this page that the terminal changes, with what each runs then, in normal mode for Shift+Alt, and
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

## helix's keys that F12 takes

These keys of helix's run an F12 action now. Its other key, where it has one,
still runs the command.

| Mode | helix's key | Its command | Its other key |
| --- | --- | --- | --- |
| normal | Ctrl+A | `increment` | none |
| normal | Ctrl+X | `decrement` | none |
| normal | Ctrl+B | `page_up` | PgUp |
| normal | Ctrl+F | `page_down` | PgDn |
| normal | Ctrl+C | `toggle_comments` | `Space c` |
| normal | Ctrl+I | `jump_forward` | Tab |
| normal | Ctrl+S | `save_selection` | none |
| normal | Ctrl+Z | `suspend` | none |
| normal | Ctrl+W | the window menu | `Space w` |
| normal | Alt+D | `delete_selection_noyank` | Delete, `"_d` |
| normal | Alt+E | `move_parent_node_end` | none |
| normal | Alt+Up | `expand_selection` | Alt+O |
| normal | Alt+Down | `shrink_selection` | Alt+I |
| normal | Alt+Left | `select_prev_sibling` | Alt+P |
| normal | Alt+Right | `select_next_sibling` | Alt+N |
| normal | Alt+: | `ensure_selections_forward` | none |
| normal | Alt+\_ | `merge_consecutive_selections` | none |
| normal | Shift+Alt+I; Shift+Alt+Down | `select_all_children` | none |
| insert | Ctrl+R | `insert_register` | none |
| insert | Ctrl+K | `kill_to_line_end` | none |
| insert | Ctrl+S | `commit_undo_checkpoint` | none |
| insert | Ctrl+W | `delete_word_backward` | Alt+Backspace |
| insert | Alt+D | `delete_word_forward` | Alt+Delete |
| insert | Ctrl+J | `insert_newline` | Enter |
| insert | Ctrl+X | `completion` | none: F12's Ctrl+Space and Ctrl+J complete |

## The actions helix has

The last column is helix's own key for the same command, for where F12's doesn't arrive.

### Editing

| F12 action | F12's keys | helix command | helix's own key | Note |
| --- | --- | --- | --- | --- |
| Undo (`$Undo`) | Ctrl+Z | `undo` | `u` |  |
| Undo, in the editor (`Editor Undo`) | Ctrl+Z | `undo` | `u` |  |
| Redo (`$Redo`) | Ctrl+Y; Shift+Ctrl+Z | `redo` | `U` |  |
| Redo, in the editor (`Editor Redo`) | none | `redo` | `U` | F12 has only Cmd+Y and Shift+Cmd+Z |
| Copy (`$Copy`) | Ctrl+C | `yank_to_clipboard` | `Space y` | normal mode alone |
| Copy, in the editor (`Editor Copy`) | Ctrl+C | `yank_to_clipboard` | `Space y` | normal mode alone |
| Cut (`$Cut`) | Ctrl+X | `yank_to_clipboard`, `delete_selection_noyank` | `Space y`, then `"_d` | normal mode alone |
| Cut, in the editor (`Editor Cut`) | none | `yank_to_clipboard`, `delete_selection_noyank` | `Space y`, then `"_d` | F12 has only Cmd+X |
| Paste (`$Paste`) | Ctrl+V | `paste_clipboard_before` | `Space P` | at the cursor, before the selection: `Space R` pastes over it; in insert mode, through `insert_register` |
| Paste, in the editor (`Editor Paste`) | Ctrl+V | `paste_clipboard_before` | `Space P` | at the cursor, before the selection: `Space R` pastes over it; in insert mode, through `insert_register` |
| Delete (`$Delete`) | Delete | `delete_selection_noyank` | `"_d` | normal mode alone: insert mode's Delete deletes the next character |
| Start new line (`EditorStartNewLine`) | Shift+Ctrl+Enter | `open_below` | `o` |  |
| Move line up (`MoveLineUp`) | Alt+Up | `extend_to_line_bounds`, `delete_selection`, `move_line_up`, `paste_before` | `X d k P` | a sequence of select mode, through register `m` |
| Move line down (`MoveLineDown`) | Alt+Down | `extend_to_line_bounds`, `delete_selection`, `paste_after` | `X d p` | a sequence of select mode, through register `m` |
| Comment line (`CommentByLineComment`) | Ctrl+Alt+/; Ctrl+K, C; Ctrl+K, Ctrl+C; Ctrl+K, Ctrl+/; Ctrl+K, Ctrl+U | `toggle_line_comments` | `Space c` |  |
| Comment with block comment (`CommentByBlockComment`) | Shift+Ctrl+/ | `toggle_block_comments` | `Space C` |  |
| Reformat code (`ReformatCode`) | Ctrl+Alt+Enter; Ctrl+K, F; Ctrl+K, Ctrl+F; Ctrl+K, D; Ctrl+K, Ctrl+D | `:format` | `:format` | the whole file: gopls formats no range |
| Reformat file (`ShowReformatFileDialog`) | Shift+Ctrl+Alt+L | `:format` | `:format` |  |
| Go fmt file (`GoFmtFileAction`) | Shift+Ctrl+Alt+F | `:format` | `:format` |  |
| Surround with live template (`SurroundWithLiveTemplate`) | Ctrl+K, S; Ctrl+K, Ctrl+S | `surround_add` | `m s` |  |
| Unwrap (`Unwrap`) | Shift+Ctrl+Delete | `surround_delete` | `m d` |  |
| Save (`SaveDocument`) | Ctrl+S | `:write` | `:w` |  |
| Save all (`SaveAll`) | Shift+Ctrl+S | `:write-all` | `:wa` |  |

### Completion

| F12 action | F12's keys | helix command | helix's own key | Note |
| --- | --- | --- | --- | --- |
| Completion (`CodeCompletion`) | Ctrl+Space; Ctrl+J | `completion` | none | insert mode alone |
| Second basic completion (`ClassNameCompletion`) | Shift+Alt+Space | `completion` | none | insert mode alone, as Alt+Space: helix drops Shift from a character key |
| Type-matching completion (`SmartTypeCompletion`) | Ctrl+Alt+Space | `completion` | none | insert mode alone |
| Cyclic expand word (`HippieCompletion`) | Alt+/ | `completion` | none | insert mode alone |
| Parameter info (`ParameterInfo`) | Shift+Ctrl+Space | none | none | not bound: Shift+Ctrl+Space arrives as Ctrl+Space, Completion; helix shows a call's parameters as it is typed |

### Selection

| F12 action | F12's keys | helix command | helix's own key | Note |
| --- | --- | --- | --- | --- |
| Select all (`$SelectAll`) | Ctrl+A | `select_all` | `%` | leaves insert mode first |
| Select all, in the editor (`Editor SelectAll`) | Ctrl+A | `select_all` | `%` | leaves insert mode first |
| Extend selection (`EditorSelectWord`) | Shift+Alt+=; Ctrl+W | `expand_selection` | Alt+O | leaves insert mode first |
| Extend selection (`SmartSelect`) | Shift+Alt+=; Ctrl+W | `expand_selection` | Alt+O | leaves insert mode first |
| Shrink selection (`EditorUnSelectWord`) | Shift+Alt+-; Shift+Ctrl+W | `shrink_selection` | Alt+I | leaves insert mode first |
| Left with selection (`EditorLeftWithSelection`) | Shift+Left | `extend_char_left` | `v h` | leaves insert mode first |
| Right with selection (`EditorRightWithSelection`) | Shift+Right | `extend_char_right` | `v l` | leaves insert mode first |
| Code block start with selection (`EditorCodeBlockStartWithSelection`) | Shift+Ctrl+Alt+\[ | `extend_parent_node_start` | `v`, then Alt+B | leaves insert mode first |
| Code block end with selection (`EditorCodeBlockEndWithSelection`) | Shift+Ctrl+Alt+\] | `extend_parent_node_end` | `v`, then Alt+E | leaves insert mode first |
| Clone caret above (`CloneCaretAboveWithVirtualSpace`) | Shift+Alt+Up | `copy_selection_on_prev_line` | Alt+C |  |
| Clone caret below (`CloneCaretBelowWithVirtualSpace`) | Shift+Alt+Down | `copy_selection_on_next_line` | `C` |  |
| Add selection for next occurrence (`SelectNextOccurrence`) | Shift+Alt+. | `search_selection_detect_word_boundaries`, `extend_search_next` | `*`, then `v n` | adds the next match of the selection; select the word first; leaves insert mode first |
| Unselect occurrence (`UnselectPreviousOccurrence`) | Shift+Alt+, | `remove_primary_selection` | Alt+, | leaves insert mode first |
| Select all occurrences (`SelectAllOccurrences`) | Shift+Alt+; | `select_references_to_symbol_under_cursor` | `Space h` | leaves insert mode first |
| Select containing declaration (`ReSharperSelectContainingDeclaration`) | Shift+Alt+\]; Shift+Ctrl+\[ | macro `maf` | `m a f` | selects around the function |

### Navigation

| F12 action | F12's keys | helix command | helix's own key | Note |
| --- | --- | --- | --- | --- |
| Go to declaration (`GotoDeclaration`) | Alt+D | `goto_definition` | `g d` |  |
| Go to implementation (`GotoImplementation`) | Ctrl+Alt+D | `goto_implementation` | `g i` |  |
| Quick definition (`QuickImplementations`) | Alt+F12 | `goto_implementation` | `g i` | jumps, with no popup |
| Go to type declaration (`GotoTypeDeclaration`) | Shift+Ctrl+F11 | `goto_type_definition` | `g y` |  |
| Find usages (`FindUsages`) | Shift+F12 | `goto_reference` | `g r` |  |
| Show usages (`ShowUsages`) | Shift+Alt+F12 | `goto_reference` | `g r` | a picker, with no popup |
| Find usages settings (`ShowSettingsAndFindUsages`) | Shift+Ctrl+Alt+F12 | `goto_reference` | `g r` |  |
| Back (`Back`) | Ctrl+- | `jump_backward` | Ctrl+O |  |
| Forward (`Forward`) | Ctrl+= | `jump_forward` | Tab | not the mouse's button 5 |
| Last edit location (`JumpToLastChange`) | Shift+Ctrl+Backspace | `goto_last_modification` | `g .` |  |
| Matching brace (`EditorMatchBrace`) | Ctrl+\] | `match_brackets` | `m m` |  |
| Code block start (`EditorCodeBlockStart`) | Shift+Alt+\[ | `move_parent_node_start` | Alt+B | the start of the syntax node around the cursor |
| Containing declaration (`ReSharperGotoContainingDeclaration`) | Ctrl+\[ | macro `maf<A-;>;` | `m a f`, then Alt+; and `;` | the start of the function around the cursor |
| Scroll to center (`EditorScrollToCenter`) | Ctrl+M, C; Ctrl+M, Ctrl+C | `align_view_center` | `z z` |  |
| Next error (`GotoNextError`) | Alt+PgDn | `goto_next_diag` | `]d` |  |
| Previous error (`GotoPreviousError`) | Alt+PgUp | `goto_prev_diag` | `[d` |  |
| Next problem file (`SwitcherNextProblemFallback`) | Alt+PgDn | `goto_next_diag` | `]d` |  |
| Quick documentation (`QuickJavaDoc`) | Ctrl+K, I; Ctrl+K, Ctrl+I; Shift+Ctrl+F1 | `hover` | `Space k` |  |

### Search

| F12 action | F12's keys | helix command | helix's own key | Note |
| --- | --- | --- | --- | --- |
| Find (`Find`) | Ctrl+F | `search` | `/` | leaves insert mode first |
| Find next (`FindNext`) | Shift+Ctrl+Down; Shift+Alt+/ | `search_next` | `n` | leaves insert mode first |
| Find in files (`FindInPath`) | Shift+Ctrl+F | `global_search` | `Space /` |  |
| Go to file (`GotoFile`) | Shift+Ctrl+T | `file_picker` | `Space f` |  |
| Open (`OpenFile`) | Ctrl+O | none | none | not bound: Ctrl+O stays helix's Back, the key for it where Ctrl+- doesn't arrive |
| Search everywhere (`SearchEverywhere`) | Ctrl+, | `file_picker` | `Space f` | files alone |
| Go to symbol (`GotoSymbol`) | Shift+Ctrl+Alt+T | `workspace_symbol_picker` | `Space S` |  |
| File structure (`FileStructurePopup`) | Alt+\\ | `symbol_picker` | `Space s` |  |
| Go to action (`GotoAction`) | Ctrl+Q | `command_palette` | `Space ?` |  |

### Code

| F12 action | F12's keys | helix command | helix's own key | Note |
| --- | --- | --- | --- | --- |
| Show intention actions (`ShowIntentionActions`) | Alt+Enter; Shift+Enter; Ctrl+Enter | `code_action` | `Space a` |  |
| Refactor this (`Refactorings.QuickListPopupAction`) | Ctrl+T | `code_action` | `Space a` | the server's code actions |
| Rename (`RenameElement`) | Ctrl+R, R | `rename_symbol` | `Space r` |  |
| Change signature (`ChangeSignature`) | Ctrl+R, S | `code_action` | `Space a` | the server's code actions |
| Extract method (`ExtractMethod`) | Ctrl+R, M | `code_action` | `Space a` | the server's code actions |
| Introduce constant (`IntroduceConstant`) | Ctrl+Alt+C | `code_action` | `Space a` | the server's code actions |
| Generate (`Generate`) | Alt+Insert | `code_action` | `Space a` | the server's code actions |
| Implement methods (`ImplementMethods`) | Shift+Ctrl+I | `code_action` | `Space a` | the server's code actions |
| Toggle inlay hints (`ToggleInlayHintsGloballyAction`) | Ctrl+I | `:toggle lsp.display-inlay-hints` | `:toggle lsp.display-inlay-hints` |  |

### Problems

| F12 action | F12's keys | helix command | helix's own key | Note |
| --- | --- | --- | --- | --- |
| Inspect code with editor settings (`CodeInspection.OnEditor`) | Shift+Alt+I | `diagnostics_picker` | `Space d` |  |
| Inspect code (`InspectCode`) | Alt+F11 | `workspace_diagnostics_picker` | `Space D` |  |
| Errors in solution (`ActivateErrorsInSolutionToolWindow`) | Ctrl+Alt+2 | `workspace_diagnostics_picker` | `Space D` |  |
| Inspection results (`ActivateInspectionResultsToolWindow`) | Ctrl+Alt+4; Ctrl+Alt+V | `workspace_diagnostics_picker` | `Space D` |  |
| Next error in solution (`ReSharperGotoNextErrorInSolution`) | Shift+Alt+PgDn; Shift+Ctrl+F12 | `workspace_diagnostics_picker` | `Space D` |  |
| Previous error in solution (`ReSharperGotoPrevErrorInSolution`) | Shift+Alt+F2; Shift+Alt+PgUp | `workspace_diagnostics_picker` | `Space D` |  |

### Files, tabs and splits

| F12 action | F12's keys | helix command | helix's own key | Note |
| --- | --- | --- | --- | --- |
| Project (`ActivateProjectToolWindow`) | F1; Alt+1 | `file_explorer` | `Space e` |  |
| Locate in solution view (`LocateInSolutionView`) | Shift+Alt+L | `file_explorer_in_current_buffer_directory` | `Space E` |  |
| Next tab (`NextTab`) | Ctrl+Alt+PgDn; Alt+Right | `goto_next_buffer` | `g n` | buffers |
| Previous tab (`PreviousTab`) | Ctrl+Alt+PgUp; Alt+Left | `goto_previous_buffer` | `g p` | buffers |
| Switcher (`Switcher`) | Shift+Ctrl+Tab | `buffer_picker` | `Space b` | buffers |
| Next splitter (`NextSplitter`) | Ctrl+End; Shift+Alt+Right | `rotate_view` | `Space w w` |  |
| Previous splitter (`PrevSplitter`) | Ctrl+Home; Shift+Alt+Left | `rotate_view_reverse` | none |  |
| Split right (`SplitVertically`) | Shift+Ctrl+Alt+R | `vsplit` | `Space w v` |  |

### Version control

| F12 action | F12's keys | helix command | helix's own key | Note |
| --- | --- | --- | --- | --- |
| Commit (`ActivateCommitToolWindow`) | F5; Alt+5 | `changed_file_picker` | `Space g` |  |
| Version control (`ActivateVersionControlToolWindow`) | F6; Alt+6 | `changed_file_picker` | `Space g` |  |
| Next difference (`NextDiff`) | F8 | `goto_next_change` | `]g` |  |
| Next change (`VcsShowNextChangeMarker`) | Shift+Ctrl+Alt+N | `goto_next_change` | `]g` |  |
| Previous difference (`PreviousDiff`) | Shift+F8; Shift+F7 | `goto_prev_change` | `[g` |  |
| Previous change (`VcsShowPrevChangeMarker`) | Shift+Ctrl+Alt+P | `goto_prev_change` | `[g` |  |
| Rollback lines (`Vcs.RollbackChangedLines`) | Ctrl+Alt+Z | `:reset-diff-change` | `:reset-diff-change` |  |

### Debugger

| F12 action | F12's keys | helix command | helix's own key | Note |
| --- | --- | --- | --- | --- |
| Toggle line breakpoint (`ToggleLineBreakpoint`) | Ctrl+B | `dap_toggle_breakpoint` | `Space G b` |  |
| Edit breakpoint (`EditBreakpoint`) | Alt+F9; Ctrl+Alt+B | `dap_edit_condition` | `Space G`, then Ctrl+C |  |
| Debug (`Debug`) | Ctrl+F5 | `dap_launch` | `Space G l` |  |
| Resume program (`Resume`) | Shift+Ctrl+F5 | `dap_continue` | `Space G c` |  |
| Pause program (`Pause`) | Ctrl+Pause; Shift+Ctrl+P | `dap_pause` | `Space G h` |  |
| Stop (`Stop`) | Alt+F5 | `dap_terminate` | `Space G t` |  |
| Step over (`StepOver`) | Alt+W | `dap_next` | `Space G n` |  |
| Step into (`StepInto`) | Alt+E | `dap_step_in` | `Space G i` |  |
| Step out (`StepOut`) | Alt+Q | `dap_step_out` | `Space G o` |  |

## The actions helix lacks

helix has no command for these, and `config.toml` binds them no key. Some share
a key with an action above.

| F12 action | F12's keys |
| --- | --- |
| `AIAssistant.Chat.AIPopupChat` | Shift+Ctrl+Alt+\\ |
| `AIAssistant.Chat.SendActions.NewLine` | Shift+Enter |
| `AIAssistant.Chat.SendActions.SendToNewChat` | Shift+Ctrl+Enter |
| `AIAssistant.Chat.Toggle.Chat.Mode` | Ctrl+Alt+/; Ctrl+Alt+Num /; Ctrl+K, C; Ctrl+K, Ctrl+C; Ctrl+K, Ctrl+/; Ctrl+K, Ctrl+U |
| `AIAssistant.CodeGeneration.Actions.ShowIntentionActions` | Alt+Enter; Shift+Enter; Ctrl+Enter |
| `AIAssistant.CodeGeneration.Actions.Specify` | Ctrl+/ |
| `AIAssistant.Editor.AskAiAssistantInEditor` | Ctrl+/ |
| `AIAssistant.ToolWindow.Chat.Cancel.Selection` | Esc |
| `AIAssistant.ToolWindow.Chat.Copy.Chameleon` | Ctrl+C |
| `AIAssistant.ToolWindow.Chat.Custom.Select.All` | Ctrl+A |
| `AIAssistant.ToolWindow.Chat.Delete` | Delete |
| `AIAssistant.ToolWindow.Chat.Find` | Ctrl+F |
| `AIAssistant.ToolWindow.NewChatAction` | Ctrl+N; Alt+Insert |
| `AIAssistant.ToolWindow.NewChatActionAlt` | Ctrl+N; Alt+Insert |
| `AIAssistant.ToolWindow.RenameDialog` | Ctrl+R, R |
| `ActivateAIAssistantToolWindow` | F7; Alt+7 |
| `ActivateBuildToolWindow` | F10 |
| `ActivateNuGetToolWindow` | F4; Alt+4 |
| `ActivateTerminalToolWindow` | F2; Alt+2 |
| `ActivateTestsToolWindow` | F3 |
| `ActivateUnitTestsToolWindow` | F3; Alt+3 |
| `AddRiderItem` | Shift+Alt+A |
| `Arrangement.Alias.Rule.Remove` | Delete |
| `Arrangement.Rule.Group.Condition.Move.Up` | Alt+Up |
| `Arrangement.Rule.Match.Condition.Move.Down` | Alt+Down |
| `Arrangement.Rule.Match.Condition.Move.Up` | Alt+Up |
| `Arrangement.Rule.Remove` | Delete |
| `AssemblyDiffAction` | Ctrl+D |
| `BookmarksView.Delete` | Delete |
| `BuildWholeSolutionAction` | Shift+Ctrl+B |
| `CIDR.Debugger.DisassembleFrame` | Ctrl+D |
| `CIDR.Debugger.ImageViewerActions.CopyImage` | Ctrl+C |
| `CallHierarchy` | Ctrl+K, T; Ctrl+K, Ctrl+T |
| `CallHierarchy.BaseOnThisMethod` | Ctrl+K, T; Ctrl+K, Ctrl+T |
| `CancelBuildAction` | Ctrl+F9; Ctrl+Cancel |
| `ChangesView.AddUnversioned` | Ctrl+Alt+A |
| `ChangesView.GroupBy.Directory` | Ctrl+Alt+P |
| `ChangesView.Move` | Shift+Alt+M |
| `ChangesView.Revert` | Ctrl+Alt+Z |
| `ChangesView.SetDefault` | Ctrl+Space |
| `ChangesView.ShelveSilently` | Shift+Ctrl+H |
| `ChangesView.ToggleCommitUi` | Ctrl+Alt+K |
| `ChangesView.UnshelveSilently` | Ctrl+Alt+U |
| `CheckinProject` | Ctrl+Alt+K |
| `ChooseDebugConfiguration` | Shift+Alt+F9 |
| `Code.Review.Editor.New.Comment` | Shift+Ctrl+X |
| `CodeFloatingToolbar.GotoNextMenu` | Tab |
| `CodeFloatingToolbar.GotoPrevMenu` | Shift+Tab |
| `CodeReview.NextComment` | F8 |
| `CodeReview.PreviousComment` | Shift+F8 |
| `CollapseAllRegions` | Ctrl+M, A; Ctrl+M, Ctrl+A |
| `CollapseExpandableComponent` | Ctrl+Num -; Ctrl+- |
| `CollapseRegion` | Ctrl+M, S; Ctrl+M, Ctrl+S |
| `CollapseSelection` | Ctrl+M, H; Ctrl+M, Ctrl+H; Ctrl+M, U; Ctrl+M, Ctrl+U |
| `Compare.SameVersion` | Ctrl+D |
| `CompareTwoFiles` | Ctrl+D |
| `Compile` | Ctrl+F7 |
| `CompileFile` | Ctrl+F7 |
| `Console.DbmsOutput` | Ctrl+B |
| `Console.Jdbc.Cancel` | Alt+F5 |
| `Console.Jdbc.Execute` | Ctrl+Enter |
| `Console.TableResult.AddColumn` | Shift+Alt+Insert |
| `Console.TableResult.AddRow` | Alt+Insert |
| `Console.TableResult.CloneColumn` | Shift+Ctrl+Alt+D |
| `Console.TableResult.CloneRow` | Ctrl+D |
| `Console.TableResult.ColumnSortAsc` | Alt+Up |
| `Console.TableResult.ColumnSortDesc` | Alt+Down |
| `Console.TableResult.ColumnSortReset` | Shift+Ctrl+Alt+Backspace |
| `Console.TableResult.ColumnsList` | Alt+\\ |
| `Console.TableResult.CompareCells` | Shift+Ctrl+D |
| `Console.TableResult.Copy` | Ctrl+C |
| `Console.TableResult.DeleteColumns` | Shift+Alt+Delete |
| `Console.TableResult.EditFilterCriteria` | Shift+Ctrl+Alt+F |
| `Console.TableResult.EditValue` | Enter; Alt+Enter |
| `Console.TableResult.FindInGrid` | Ctrl+F |
| `Console.TableResult.GotoReferencedResult` | Enter; Alt+Enter; mouse double-click |
| `Console.TableResult.GotoReferencingResult` | Alt+D |
| `Console.TableResult.MaximizeEditingCell` | Shift+Ctrl+Alt+M |
| `Console.TableResult.NavigateAction` | Alt+S |
| `Console.TableResult.OpenLocalFileAction` | Alt+S |
| `Console.TableResult.OpenWebUrlAction` | Alt+S |
| `Console.TableResult.RevertSelected` | Ctrl+Alt+Z |
| `Console.TableResult.SelectColumn` | Ctrl+Space |
| `Console.TableResult.SelectRow` | Shift+Space |
| `Console.TableResult.SetNull` | Ctrl+Alt+N |
| `Console.TableResult.ShowRecordView` | Shift+Ctrl+Enter |
| `Console.TableResult.Submit` | Ctrl+Enter |
| `Console.TableResult.SubmitAndCommit` | Shift+Ctrl+Alt+Enter |
| `Console.Transaction.Commit` | Shift+Ctrl+Alt+Enter |
| `CopyReference` | Shift+Ctrl+Alt+C |
| `CppIncludesHierarchy` | Shift+Alt+H |
| `DatabaseView.CopyAction` | Ctrl+D |
| `DatabaseView.CopyDdlAction` | Shift+Ctrl+Alt+G |
| `DatabaseView.DataSourceCopyAction` | Ctrl+C |
| `DatabaseView.DataSourceCutAction` | Ctrl+X |
| `DatabaseView.DataSourcePasteAction` | Ctrl+V |
| `DatabaseView.Ddl.AlterObject` | Ctrl+R, S |
| `DatabaseView.DropAction` | Delete |
| `DatabaseView.FullTextSearch` | Shift+Ctrl+Alt+F |
| `DatabaseView.OpenDdlInConsole` | Shift+Ctrl+Alt+B |
| `DatabaseView.ShowDiff` | Ctrl+D |
| `DatabaseView.SqlGenerator` | Ctrl+Alt+G |
| `DatabaseView.Tools` | Alt+Enter; Shift+Enter; Ctrl+Enter |
| `DatabaseView.Tools.RevertChanges` | Ctrl+Alt+Z |
| `DatabaseView.Tools.SubmitChanges` | Ctrl+Alt+K |
| `Debugger.MarkObject` | Ctrl+K, K; Ctrl+K, Ctrl+K |
| `DecreaseColumnWidth` | Ctrl+Alt+Left |
| `Diagram.DeleteSelection` | Delete |
| `Diagram.DeselectAll` | Ctrl+Alt+A |
| `Diagram.SelectAll` | Ctrl+A |
| `Diff.FocusOppositePane` | Ctrl+\\, Ctrl+Tab; Ctrl+\\, Tab; Shift+Ctrl+Tab |
| `Diff.ShowDiff` | Ctrl+D |
| `Docker.RemoteServers.StopComposeApp` | Alt+F5 |
| `Docker.RemoteServers.StopDeploy` | Alt+F5 |
| `Documentation.EditSource` | Alt+S |
| `DomCollectionControl.Add` | Insert |
| `DomCollectionControl.Edit` | Alt+S |
| `DomCollectionControl.Remove` | Delete |
| `DownloadBackendFileToClient` | Shift+Ctrl+D |
| `DumpLookupElementWeights` | Shift+Ctrl+Alt+W |
| `DumpMLCompletionFeatures` | Shift+Ctrl+Alt+9 |
| `EditPropertiesAction` | Alt+Enter; Shift+Enter; Ctrl+Enter |
| `EditSource` | Alt+S |
| `EditorAddOrRemoveCaret` | mouse Ctrl+Alt+click |
| `EditorCreateRectangularSelection` | mouse Shift+Alt+click |
| `EditorCreateRectangularSelectionOnMouseDrag` | mouse Alt+click; mouse Shift+Alt+click |
| `EditorDecreaseFontSize` | Shift+Ctrl+-; mouse Ctrl+wheel down |
| `EditorIncreaseFontSize` | Shift+Ctrl+=; mouse Ctrl+wheel up |
| `EditorResetFontSizeGlobal` | Shift+Ctrl+\\ |
| `EmmetNextEditPoint` | Shift+Alt+\] |
| `EmmetPreviousEditPoint` | Shift+Alt+\[ |
| `EnableDaemon` | Shift+Ctrl+Alt+8 |
| `EvaluateExpression` | Alt+F |
| `ExpandAll` | Ctrl+Num + |
| `ExpandAllRegions` | Ctrl+M, X; Ctrl+M, Ctrl+X |
| `ExpandCollapseToggleAction` | Ctrl+M, M; Ctrl+M, Ctrl+M |
| `ExpandExpandableComponent` | Ctrl+Num + |
| `ExpandRegion` | Ctrl+M, E; Ctrl+M, Ctrl+E |
| `ExportToTextFile` | Alt+O |
| `ExternalJavaDoc` | Shift+F1 |
| `ExternalSystem.CollapseAll` | Ctrl+Num -; Ctrl+- |
| `ExternalSystem.DetachProject` | Delete |
| `ExternalSystem.ExpandAll` | Ctrl+Num + |
| `FileChooser.Delete` | Delete |
| `FileChooser.GoBackward` | Ctrl+-; Ctrl+Num - |
| `FileChooser.GoToParent` | Backspace |
| `FileChooser.GoToRoot` | Ctrl+\\ |
| `FileChooser.GotoDesktop` | Ctrl+D |
| `FileChooser.NewFolder` | Alt+Insert; Ctrl+N |
| `FileChooser.TogglePathBar` | Ctrl+P |
| `ForceRunToCursor` | Ctrl+Alt+R |
| `ForceStepInto` | Ctrl+Alt+E |
| `ForceStepOver` | Ctrl+Alt+W |
| `Generate.Missing.Members.ES6` | Shift+Ctrl+I |
| `Generate.Missing.Members.TypeScript` | Shift+Ctrl+I |
| `Git.Add` | Ctrl+Alt+A |
| `Git.Branches` | Alt+X |
| `Git.Commit.And.Push.Executor` | Alt+C |
| `Git.Commit.Stage` | Ctrl+Alt+K |
| `Git.CreateNewBranch` | Ctrl+Alt+N |
| `Git.CreateNewBranch.FromCommit` | Ctrl+Alt+N |
| `Git.Fetch` | Alt+Z |
| `Git.Log.Branches.Navigate.Log.To.Selected.Branch` | Alt+F1 |
| `Git.New.Branch.In.Log` | Ctrl+Alt+N |
| `Git.ShowBranches` | Alt+X |
| `Git.Stage.Add` | Ctrl+Alt+A |
| `Git.Stage.Reset` | Ctrl+Alt+Z |
| `Git.Stage.Revert` | Ctrl+Alt+Z |
| `Git.Stash.Drop` | Delete |
| `Git.WorkingTrees.Refresh` | Ctrl+Alt+Y |
| `Git.WorkingTrees.Remove` | Delete |
| `GoCallHierarchyPopupMenu.BaseOnThisDeclaration` | Ctrl+K, T; Ctrl+K, Ctrl+T |
| `GoGenerateFileAction` | Ctrl+Alt+G |
| `GoOpenSettings` | Alt+S |
| `GoShareInPlaygroundAction` | Shift+Ctrl+Alt+S |
| `GoToControllerOrView` | Alt+O; Ctrl+K, O; Ctrl+K, Ctrl+O |
| `GotoCustomRegion` | Ctrl+Alt+. |
| `GotoNextBookmark` | Ctrl+K, N; Ctrl+K, Ctrl+N |
| `GotoNextElementUnderCaretUsage` | Shift+Ctrl+Down |
| `GotoPrevElementUnderCaretUsage` | Shift+Ctrl+Up |
| `GotoPreviousBookmark` | Ctrl+K, P; Ctrl+K, Ctrl+P |
| `GotoRelated` | Ctrl+Alt+F7 |
| `GotoSuperMethod` | Alt+Home |
| `Graph.Delete` | Delete |
| `Graph.ZoomIn` | Num + |
| `Graph.ZoomOut` | Num - |
| `HTTPClient.PreviewHtml.NavigateBack` | Left |
| `HTTPClient.PreviewHtml.NavigateForward` | Right |
| `HelpTopics` | Ctrl+F1 |
| `Hg.Commit.And.Push.Executor` | Shift+Ctrl+Alt+K |
| `HideAllWindows` | Shift+Esc |
| `Images.EditExternally` | Ctrl+Alt+F4 |
| `Images.Editor.ToggleGrid` | Ctrl+' |
| `Images.Editor.ZoomIn` | Ctrl+Num + |
| `Images.Editor.ZoomOut` | Ctrl+Num -; Ctrl+- |
| `Images.Thumbnails.Hide` | Ctrl+F4 |
| `Images.Thumbnails.ToggleRecursive` | Alt+Num \* |
| `Images.Thumbnails.UpFolder` | Backspace |
| `IncreaseColumnWidth` | Ctrl+Alt+Right |
| `InlinePromptGenerateCodeAction` | Tab |
| `InsertLiveTemplate` | Ctrl+K, X; Ctrl+K, Ctrl+X |
| `InsertNextEditAction` | Tab |
| `InspectThis` | Shift+Ctrl+Alt+A |
| `JavaScript.ShowComponentUsages` | Shift+Ctrl+D |
| `Jdbc.OpenConsole.New` | Shift+Ctrl+Q |
| `Jdbc.OpenConsole.New.Generate` | Shift+Ctrl+Q |
| `Jdbc.OpenConsole.New.InPath` | Shift+Ctrl+Alt+Q |
| `Jdbc.OpenConsole.Scratch` | Shift+Ctrl+Alt+Q |
| `Jdbc.OpenEditor.Console` | Alt+S |
| `Jdbc.OpenEditor.DDL` | Alt+D |
| `Jdbc.OpenEditor.Grid.DDL` | Ctrl+Alt+F7 |
| `JsonPathExportEvaluateResultAction` | Alt+O |
| `JumpToLastWindow` | Ctrl+Alt+Backspace; Shift+Alt+F6 |
| `JumpToStatement` | Shift+Ctrl+F10 |
| `List-selectNextColumnExtendSelection` | Shift+Right |
| `MainMenuButton.ShowMenu` | Alt+. |
| `MaintenanceAction` | Shift+Ctrl+Alt+/ |
| `Markdown.Insert` | Alt+Insert |
| `Markdown.Preview.DecreaseFontSize` | Ctrl+Num -; Ctrl+- |
| `Markdown.Styling.CreateLink` | Shift+Ctrl+U |
| `MiniAiChat.History` | Alt+Down |
| `ModifyObject` | Ctrl+R, S |
| `NewElement` | Ctrl+N; Alt+Insert |
| `NewElementSamePlace` | Shift+Ctrl+A |
| `NewRiderProject` | Shift+Ctrl+N |
| `NewScratchFile` | Shift+Ctrl+Alt+Insert |
| `NextOccurence` | F8 |
| `OasEndpointsSidePanelSaveAction` | Alt+O |
| `OpenInRightSplit` | mouse Alt+double-click |
| `OpenWinFormsDesignerAction` | Shift+F7 |
| `Orchide.GotoInventory` | Shift+Ctrl+O, I |
| `PCFindUsagesAction` | Shift+F12 |
| `PCNavigateToSource` | Alt+S |
| `ParameterInfo.GoToPreviousSignature` | Shift+Ctrl+P |
| `PasteMultiple` | Shift+Ctrl+V; Shift+Ctrl+Insert |
| `PopupHector` | Shift+Ctrl+Alt+H |
| `PreviousOccurence` | Shift+F8 |
| `PublishGroup.UploadTo` | Shift+Ctrl+Alt+X |
| `ReSharperNavigateTo` | Alt+\`; Alt+Dead grave |
| `ReformatWithPrettierAction` | Shift+Ctrl+Alt+P |
| `RemoteHostView.EditRemoteFile` | Enter |
| `RemoteHostView.Rename` | Ctrl+R, R |
| `Replace` | Ctrl+H |
| `ReplaceInPath` | Shift+Ctrl+H |
| `ResetColumnsWidth` | Ctrl+Alt+Up |
| `ResizeToolWindowDown` | Ctrl+Alt+Down |
| `ResizeToolWindowLeft` | Ctrl+Alt+Left |
| `ResizeToolWindowRight` | Ctrl+Alt+Right |
| `ResizeToolWindowUp` | Ctrl+Alt+Up |
| `Rider.Plugins.FSharp.StartFsi` | Ctrl+Alt+F |
| `RiderCodeLens.ShowMore` | Ctrl+K, Ctrl+\`; Ctrl+K, \` |
| `RiderCollapseToDefinitions` | Ctrl+M, O; Ctrl+M, Ctrl+O |
| `RiderDebuggerApplyEncChagnes` | Alt+F10 |
| `RiderDotCoverCoveringTestsPopupCoverAllTestsAction` | Ctrl+K |
| `RiderDotCoverCoveringTestsPopupCoverSelectedTestsAction` | Ctrl+C |
| `RiderDotCoverCoveringTestsPopupDebugSelectedTestsAction` | Ctrl+D |
| `RiderDotCoverCoveringTestsPopupRunAllTestsAction` | Ctrl+R |
| `RiderDotCoverCoveringTestsPopupRunSelectedTestsAction` | Ctrl+S |
| `RiderDotCoverUnitTestCoverSolutionAction` | Ctrl+U, K; Ctrl+U, Ctrl+K |
| `RiderEditSource` | Alt+S |
| `RiderGenerateUnitTestAction` | Ctrl+U, C; Ctrl+U, Ctrl+C |
| `RiderGoToLinkedTypesAction` | Ctrl+U, G; Ctrl+U, Ctrl+G |
| `RiderNuGetContextPopupAction` | Alt+Enter; Shift+Enter; Ctrl+Enter |
| `RiderNuGetCopyPackageNameAction` | Ctrl+C |
| `RiderNuGetQuickDocAction` | Ctrl+K, I; Ctrl+K, Ctrl+I; Shift+Ctrl+F1 |
| `RiderNuGetQuickListPopupAction` | Shift+Alt+N |
| `RiderNuGetToggleDependenciesExpanderAction` | Ctrl+D |
| `RiderNuGetToggleInfoExpanderAction` | Ctrl+I |
| `RiderOpenSolution` | Shift+Ctrl+O |
| `RiderProblemsViewToolsetContextPopupAction` | Alt+Enter; Shift+Enter; Ctrl+Enter |
| `RiderReattach` | Shift+Alt+P |
| `RiderRemoveAllLineBreakpoints` | Shift+Ctrl+F9 |
| `RiderUnitTestAppendTestsAction` | Ctrl+U, A; Ctrl+U, Ctrl+A |
| `RiderUnitTestAppendTestsTwAction` | Ctrl+Alt+Insert |
| `RiderUnitTestDebugContextAction` | Ctrl+U, D; Ctrl+U, Ctrl+D |
| `RiderUnitTestDebugContextTwAction` | Ctrl+D |
| `RiderUnitTestDotMemoryUnitContextAction` | Ctrl+U, M; Ctrl+U, Ctrl+M |
| `RiderUnitTestDotMemoryUnitSolutionAction` | Ctrl+U, E; Ctrl+U, Ctrl+E |
| `RiderUnitTestFocusExplorerAction` | Ctrl+Alt+U |
| `RiderUnitTestFocusSessionAction` | Ctrl+Alt+T |
| `RiderUnitTestNewSessionAction` | Ctrl+U, N; Ctrl+U, Ctrl+N |
| `RiderUnitTestNewSessionTwAction` | Shift+Alt+Insert |
| `RiderUnitTestQuickListPopupAction` | Shift+Alt+U |
| `RiderUnitTestRemoveElementsFromSessionTwAction` | Delete |
| `RiderUnitTestRepeatPreviousRunAction` | Ctrl+U, U; Ctrl+U, Ctrl+U |
| `RiderUnitTestRunContextAction` | Ctrl+U, R; Ctrl+U, Ctrl+R |
| `RiderUnitTestRunContextUntilFailAction` | Ctrl+U, W; Ctrl+U, Ctrl+W |
| `RiderUnitTestRunCurrentSessionAction` | Ctrl+U, Y; Ctrl+U, Ctrl+Y |
| `RiderUnitTestRunCurrentSessionTwAction` | Ctrl+Y |
| `RiderUnitTestRunSolutionAction` | Ctrl+U, L; Ctrl+U, Ctrl+L |
| `RiderUnitTestRunSolutionTwAction` | Ctrl+L |
| `RiderUnitTestRunTreeTwAction` | Shift+Ctrl+Enter |
| `RiderUnitTestSessionAbortAction` | Ctrl+U, S; Ctrl+U, Ctrl+S |
| `RiderUnitTestSessionRerunFailedTestsAction` | Ctrl+U, F; Ctrl+U, Ctrl+F |
| `RiderUnitTestTreeSelectionPopupAction` | Alt+Enter; Shift+Enter; Ctrl+Enter |
| `RiderUnitTestTreeTextFilterAction` | Ctrl+F |
| `Run` | Shift+F5 |
| `RunDashboard.CopyConfiguration` | Ctrl+D |
| `RunDashboard.UngroupConfigurations` | Delete |
| `RunInspection` | Shift+Ctrl+Alt+I |
| `RunJsbtTask` | Shift+Ctrl+Alt+F11 |
| `RunToCursor` | Alt+R |
| `SafeDelete` | Alt+Delete |
| `SaveAs` | Shift+Ctrl+S |
| `SchemaCompare.Cancel` | Alt+Pause |
| `SchemaCompare.Compare` | Shift+Alt+C |
| `SchemaCompare.GenerateDeployScript` | Shift+Alt+G |
| `SchemaCompare.SaveScmpFile` | Ctrl+S |
| `SchemaCompare.UpdateTarget` | Shift+Alt+U |
| `ScrollTreeToCenter` | Ctrl+M, C; Ctrl+M, Ctrl+C |
| `SendEOF` | Ctrl+D |
| `ServiceView.GroupByContributor` | Ctrl+Alt+T |
| `ServiceView.JumpToServices` | Ctrl+F2 |
| `Session.Rename` | Ctrl+R, R |
| `ShelveChanges.UnshelveWithDialog` | Shift+Ctrl+U |
| `ShowErrorDescription` | Ctrl+E |
| `ShowExecutionPoint` | Alt+Num \* |
| `ShowFilterPopup` | Ctrl+Alt+F |
| `ShowNavBar` | Ctrl+F2 |
| `ShowPopupMenu` | Menu; F13 |
| `ShowUmlDiagram` | Shift+Ctrl+Alt+U |
| `ShowUmlDiagramPopup` | Ctrl+Alt+U |
| `StretchSplitToBottom` | Ctrl+Alt+Down |
| `StretchSplitToLeft` | Ctrl+Alt+Left |
| `StretchSplitToRight` | Ctrl+Alt+Right |
| `StretchSplitToTop` | Ctrl+Alt+Up |
| `SwitchCodeToDesigner` | Shift+F7 |
| `SwitchCoverage` | Ctrl+Alt+F6 |
| `SwitchHeaderSource` | Alt+O; Ctrl+K, O; Ctrl+K, Ctrl+O |
| `Synchronize` | Ctrl+Alt+Y |
| `Table-selectNextColumnExtendSelection` | Shift+Right |
| `Table-selectPreviousColumnExtendSelection` | Shift+Left |
| `TableResult.GrowSelection` | Shift+Alt+=; Ctrl+W |
| `TableResult.SelectAllOccurrences` | Shift+Alt+; |
| `TableResult.SelectColumn` | Shift+Alt+=; Ctrl+W |
| `TableResult.SelectNextOccurrence` | Shift+Alt+. |
| `TableResult.ShrinkSelection` | Shift+Alt+-; Shift+Ctrl+W |
| `TableResult.UnselectPreviousOccurrence` | Shift+Alt+, |
| `Terminal.CopySelectedText` | Ctrl+C; Ctrl+Insert |
| `Terminal.ExpandBlockSelectionAbove` | Shift+Up |
| `Terminal.ExpandBlockSelectionBelow` | Shift+Down |
| `Terminal.GenerateCommandFromText` | Ctrl+/ |
| `Terminal.Paste` | Ctrl+V; Shift+Insert |
| `Terminal.SelectBlockAbove` | Up; Ctrl+Up |
| `Terminal.SelectBlockBelow` | Down |
| `Terminal.SelectLastBlock` | Ctrl+Up |
| `Terminal.SelectPrompt` | Ctrl+Down |
| `Terminal.ShowDocumentation` | Ctrl+K, I; Ctrl+K, Ctrl+I; Shift+Ctrl+F1 |
| `TodoViewGroupByFlattenPackage` | Ctrl+Alt+C |
| `TodoViewGroupByShowModules` | Ctrl+Alt+M |
| `TodoViewGroupByShowPackages` | Ctrl+Alt+P |
| `ToggleBookmarkWithMnemonic` | Ctrl+F11 |
| `ToggleBreakpointEnabled` | Ctrl+F9 |
| `ToggleFullScreen` | Shift+Alt+Enter |
| `ToggleRenderedDocPresentation` | Ctrl+K, V; Ctrl+K, Ctrl+V |
| `ToggleTemporaryLineBreakpoint` | Shift+Ctrl+Alt+F8 |
| `Tree-selectParentExtendSelection` | Shift+Left |
| `UML.Find` | Ctrl+F |
| `UML.ShowChanges` | Shift+Ctrl+Alt+D |
| `UML.ShowStructure` | Alt+\\ |
| `Uml.NewElement` | Ctrl+N; Alt+Insert |
| `Uml.NodeIntentions` | Alt+Enter; Shift+Enter; Ctrl+Enter |
| `Uml.ShowDiff` | Shift+Ctrl+D |
| `UpdateRunningApplication` | Ctrl+F10 |
| `UsageView.Exclude` | Delete |
| `UsageView.Include` | Insert |
| `Vcs.CombinedDiff.CaretToNextBlock` | Down; Right; PgDn |
| `Vcs.CombinedDiff.CaretToPrevBlock` | Up; Left; PgUp |
| `Vcs.CombinedDiff.ToggleCollapseBlock` | Ctrl+Esc |
| `Vcs.Commit.CloseDialog` | Esc |
| `Vcs.CopyRevisionNumberAction` | Shift+Ctrl+Alt+C |
| `Vcs.EditSource` | Alt+S |
| `Vcs.Log.GoToChild` | Left |
| `Vcs.Log.GoToParent` | Right |
| `Vcs.Log.GoToRef` | Ctrl+F |
| `Vcs.Log.ShowAllAffected` | Shift+Alt+A |
| `Vcs.Log.ShowTooltip` | Ctrl+K, I; Ctrl+K, Ctrl+I; Shift+Ctrl+F1 |
| `Vcs.MoveChangedLinesToChangelist` | Shift+Alt+M |
| `Vcs.Push` | Alt+V |
| `Vcs.ReformatCommitMessage` | Ctrl+Alt+Enter; Ctrl+K, F; Ctrl+K, Ctrl+F; Ctrl+K, D; Ctrl+K, Ctrl+D |
| `Vcs.Shelf.Drop` | Delete |
| `Vcs.ShowMessageHistory` | Ctrl+H |
| `Vcs.ToggleAmendCommitMode` | Alt+A |
| `VcsHistory.ShowAllAffected` | Shift+Alt+A |
| `Voice.ActivateVoiceAction` | Shift+Alt+E |
| `Voice.ReferenceEntityInChatAction` | Shift+Alt+A |
| `WD.UploadCurrentRemoteFileAction` | Shift+Alt+Q |
| `WebOpenInAction` | Alt+F2; Ctrl+Alt+F2 |
| `WelcomeScreen.RemoveSelected` | Delete |
| `XDebugger.AttachToProcess` | Ctrl+Alt+P |
| `XDebugger.CopyWatch` | Ctrl+D |
| `XDebugger.JumpToTypeSource` | Shift+F4 |
| `XDebugger.MoveWatchDown` | Alt+Down |
| `XDebugger.MoveWatchUp` | Alt+Up |
| `XDebugger.NewWatch` | Insert |
| `XDebugger.RemoveWatch` | Delete |
| `XPathView.Actions.Evaluate` | Ctrl+Alt+X, E |
| `XPathView.Actions.FindByExpression` | Ctrl+Alt+X, F |
| `XPathView.Actions.ShowPath` | Ctrl+Alt+X, P |
| `ZoomInIdeAction` | Shift+Ctrl+Alt+. |
| `ZoomOutIdeAction` | Shift+Ctrl+Alt+, |
| `context.clear` | Shift+Alt+X |
| `context.load` | Shift+Alt+L |
| `context.save` | Shift+Alt+S |
| `hg4idea.QGotoFromPatches` | Shift+Alt+G |
| `hg4idea.QPushAction` | Shift+Alt+P |
| `hg4idea.add` | Ctrl+Alt+A |
| `openAssertEqualsDiff` | Ctrl+D |
| `org.intellij.plugins.markdown.ui.actions.styling.InsertImageAction` | Ctrl+U |
| `org.intellij.plugins.markdown.ui.actions.styling.ToggleStrikethroughAction` | Shift+Ctrl+S |
| `sql.ExtractFunctionAction` | Ctrl+R, M |
| `sql.SelectInDatabaseView` | Shift+Alt+B |
| `tasks.open.in.browser` | Shift+Alt+B |
| `tasks.switch` | Shift+Alt+T |
