# The actions helix has

The last column is helix's own key for the same command, for where F12's doesn't arrive.

## Editing

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

## Completion

| F12 action | F12's keys | helix command | helix's own key | Note |
| --- | --- | --- | --- | --- |
| Completion (`CodeCompletion`) | Ctrl+Space; Ctrl+J | `completion` | none | insert mode alone |
| Second basic completion (`ClassNameCompletion`) | Shift+Alt+Space | `completion` | none | insert mode alone, as Alt+Space: helix drops Shift from a character key |
| Type-matching completion (`SmartTypeCompletion`) | Ctrl+Alt+Space | `completion` | none | insert mode alone |
| Cyclic expand word (`HippieCompletion`) | Alt+/ | `completion` | none | insert mode alone |
| Parameter info (`ParameterInfo`) | Shift+Ctrl+Space | none | none | not bound: Shift+Ctrl+Space arrives as Ctrl+Space, Completion; helix shows a call's parameters as one types it |

## Selection

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

## Navigation

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

## Search

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

## Code

<!-- Implement methods is an IDE action's name, which write-good takes for a wordy verb. -->
<!-- vale write-good.TooWordy = NO -->

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

<!-- vale write-good.TooWordy = YES -->

## Problems

| F12 action | F12's keys | helix command | helix's own key | Note |
| --- | --- | --- | --- | --- |
| Inspect code with editor settings (`CodeInspection.OnEditor`) | Shift+Alt+I | `diagnostics_picker` | `Space d` |  |
| Inspect code (`InspectCode`) | Alt+F11 | `workspace_diagnostics_picker` | `Space D` |  |
| Errors in solution (`ActivateErrorsInSolutionToolWindow`) | Ctrl+Alt+2 | `workspace_diagnostics_picker` | `Space D` |  |
| Inspection results (`ActivateInspectionResultsToolWindow`) | Ctrl+Alt+4; Ctrl+Alt+V | `workspace_diagnostics_picker` | `Space D` |  |
| Next error in solution (`ReSharperGotoNextErrorInSolution`) | Shift+Alt+PgDn; Shift+Ctrl+F12 | `workspace_diagnostics_picker` | `Space D` |  |
| Previous error in solution (`ReSharperGotoPrevErrorInSolution`) | Shift+Alt+F2; Shift+Alt+PgUp | `workspace_diagnostics_picker` | `Space D` |  |

## Files, tabs and splits

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

## Version control

| F12 action | F12's keys | helix command | helix's own key | Note |
| --- | --- | --- | --- | --- |
| Commit (`ActivateCommitToolWindow`) | F5; Alt+5 | `changed_file_picker` | `Space g` |  |
| Version control (`ActivateVersionControlToolWindow`) | F6; Alt+6 | `changed_file_picker` | `Space g` |  |
| Next difference (`NextDiff`) | F8 | `goto_next_change` | `]g` |  |
| Next change (`VcsShowNextChangeMarker`) | Shift+Ctrl+Alt+N | `goto_next_change` | `]g` |  |
| Previous difference (`PreviousDiff`) | Shift+F8; Shift+F7 | `goto_prev_change` | `[g` |  |
| Previous change (`VcsShowPrevChangeMarker`) | Shift+Ctrl+Alt+P | `goto_prev_change` | `[g` |  |
| Rollback lines (`Vcs.RollbackChangedLines`) | Ctrl+Alt+Z | `:reset-diff-change` | `:reset-diff-change` |  |

## Debugger

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
