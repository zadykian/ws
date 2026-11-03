# The actions the Zed keymap binds

Each row is an F12 action with keys, and what Zed does with them. The status `bind` is the same
behaviour, `approx` the closest Zed action, and `zed` that Zed's own default already does it.
F12's actions come with those of IntelliJ's default keymap that F12 leaves alone. This page covers
Editing to Folding.

## Editing

| F12 action | Keys | Zed keys | Zed action | Context | Status | Note |
| --- | --- | --- | --- | --- | --- | --- |
| `$Copy` | Cmd+C; Ctrl+C | `cmd-c`; `ctrl-c` | `editor::Copy` | `Editor` | bind | Cmd+C is Zed's already |
| `$Cut` | Ctrl+X; Cmd+X | `ctrl-x`; `cmd-x` | `editor::Cut` | `Editor` | bind |  |
| `$Delete` | Delete | `delete` | `editor::Delete` | `Editor` | zed |  |
| `$Paste` | Cmd+V; Ctrl+V | `cmd-v`; `ctrl-v` | `editor::Paste` | `Editor` | bind | Ctrl+V replaces Zed's emacs-style page down |
| `$Redo` | Ctrl+Y; Shift+Ctrl+Z; Cmd+Y; Shift+Cmd+Z | `ctrl-y`; `ctrl-shift-z`; (not `cmd-y`); `cmd-shift-z` | `editor::Redo` | `Editor` | bind | Shift+Cmd+Z is Zed's; Cmd+Y stays Zed's: the agent's Allow once and Keep, and git::StageAndNext |
| `$Undo` | Cmd+Z; Ctrl+Z | `cmd-z`; `ctrl-z` | `editor::Undo` | `Editor` | bind | Cmd+Z is Zed's already |
| `CommentByBlockComment` | Shift+Ctrl+/; Shift+Ctrl+numpad / | `ctrl-?` | `editor::ToggleBlockComments` | `Editor` | bind |  |
| `CommentByLineComment` | Ctrl+Alt+/; Ctrl+Alt+numpad /; Ctrl+K, C; Ctrl+K, Ctrl+C; Ctrl+K, Ctrl+/; Ctrl+K, Ctrl+U | `ctrl-alt-/`; `ctrl-k c`; `ctrl-k ctrl-c`; `ctrl-k ctrl-/`; `ctrl-k ctrl-u` | `editor::ToggleComments` | `Editor` | bind | Ctrl+K, Ctrl+U toggles rather than uncomments |
| `Editor Copy` | Cmd+C; Ctrl+C | `cmd-c`; `ctrl-c` | `editor::Copy` | `Editor` | bind |  |
| `Editor Cut` | Cmd+X | `cmd-x` | `editor::Cut` | `Editor` | bind | Cmd+X alone in F12; $Cut's Ctrl+X cuts in the editor too |
| `Editor Paste` | Cmd+V; Ctrl+V | `cmd-v`; `ctrl-v` | `editor::Paste` | `Editor` | bind |  |
| `Editor Redo` | Cmd+Y; Shift+Cmd+Z | (not `cmd-y`); `cmd-shift-z` | `editor::Redo` | `Editor` | bind | Cmd keys alone in F12; Cmd+Y stays Zed's, as for $Redo |
| `Editor Undo` | Cmd+Z; Ctrl+Z | `cmd-z`; `ctrl-z` | `editor::Undo` | `Editor` | bind |  |
| `EditorBackSpace` | Backspace; Shift+Backspace | `backspace`; `shift-backspace` |  | `Editor` | zed | Zed's editor keys |
| `EditorDecreaseFontSize` | Shift+Ctrl+-; mouse control button142 | `ctrl-_` | `zed::DecreaseBufferFontSize` | `Editor` | bind | mouse wheel left to Zed |
| `EditorDeleteToWordEnd` | Ctrl+Delete | `ctrl-delete` | `editor::DeleteToNextWordEnd` | `Editor` | bind |  |
| `EditorDeleteToWordStart` | Ctrl+Backspace | `ctrl-backspace` | `editor::DeleteToPreviousWordStart` | `Editor` | bind |  |
| `EditorDuplicate` | Ctrl+D | `ctrl-d` | `editor::DuplicateSelection` | `Editor` | bind | replaces Zed's emacs-style delete |
| `EditorEnter` | Enter | `enter` |  | `Editor` | zed | Zed's editor keys |
| `EditorEscape` | Esc | `escape` |  | `Editor` | zed | Zed's editor keys |
| `EditorIncreaseFontSize` | Shift+Ctrl+=; mouse control button143 | `ctrl-+` | `zed::IncreaseBufferFontSize` | `Editor` | bind |  |
| `EditorIndentSelection` | Tab | `tab` |  | `Editor` | zed | Zed's editor keys |
| `EditorJoinLines` | Shift+Ctrl+J | `ctrl-shift-j` | `editor::JoinLines` | `Editor` | bind |  |
| `EditorPasteSimple` | Shift+Ctrl+Alt+V | `ctrl-alt-shift-v` | `editor::Paste` | `Editor` | approx |  |
| `EditorResetFontSizeGlobal` | Shift+Ctrl+\ | `ctrl-\|` | `zed::ResetBufferFontSize` | `Editor` | bind |  |
| `EditorStartNewLine` | Shift+Ctrl+Enter | `ctrl-shift-enter` | `editor::NewlineBelow` | `Editor && mode == full` | bind |  |
| `EditorTab` | Tab | `tab` |  | `Editor` | zed | Zed's editor keys |
| `EditorToggleCase` | Shift+Ctrl+U | `ctrl-shift-u` | `editor::ToggleCase` | `Editor` | bind |  |
| `EditorUnindentSelection` | Shift+Tab | `shift-tab` |  | `Editor` | zed | Zed's editor keys |
| `InsertLiveTemplate` | Ctrl+K, X; Ctrl+K, Ctrl+X | `ctrl-k x`; `ctrl-k ctrl-x` | `editor::ShowCompletions` | `Editor && mode == full` | approx | Zed offers snippets among completions |
| `MoveLineDown` | Alt+Down | `alt-down` | `editor::MoveLineDown` | `Editor` | zed | Alt+Down is Zed's on macOS |
| `MoveLineUp` | Alt+Up | `alt-up` | `editor::MoveLineUp` | `Editor` | zed | Alt+Up is Zed's on macOS |
| `SaveAll` | Shift+Ctrl+S; Ctrl+Cmd+S | `ctrl-shift-s`; `ctrl-cmd-s` | `workspace::SaveAll` | `(none)` | bind | Ctrl+Cmd+S replaces Zed's ToggleWorktreeSecurity |
| `SaveDocument` | Ctrl+S; Cmd+S | `ctrl-s`; `cmd-s` | `workspace::Save` | `(none)` | bind | Cmd+S is Zed's already |
| `Unwrap` | Shift+Ctrl+Delete | `ctrl-shift-delete` | `editor::UnwrapSyntaxNode` | `Editor && mode == full` | bind | the syntax node around the cursor |
| `ZoomInIdeAction` | Shift+Ctrl+Alt+. | `ctrl-alt->` | `zed::IncreaseUiFontSize` | `(none)` | bind |  |
| `ZoomOutIdeAction` | Shift+Ctrl+Alt+, | `ctrl-alt-<` | `zed::DecreaseUiFontSize` | `(none)` | bind |  |

## Selection

| F12 action | Keys | Zed keys | Zed action | Context | Status | Note |
| --- | --- | --- | --- | --- | --- | --- |
| `$SelectAll` | Cmd+A; Ctrl+A | `cmd-a`; `ctrl-a` | `editor::SelectAll` | `Editor` | bind | Ctrl+A replaces Zed's emacs-style line start |
| `CloneCaretAboveWithVirtualSpace` | Shift+Alt+Up | `alt-shift-up` | `editor::AddSelectionAbove` | `Editor` | bind | replaces Zed's DuplicateLineUp |
| `CloneCaretBelowWithVirtualSpace` | Shift+Alt+Down | `alt-shift-down` | `editor::AddSelectionBelow` | `Editor` | bind | replaces Zed's DuplicateLineDown |
| `Editor SelectAll` | Cmd+A; Ctrl+A | `cmd-a`; `ctrl-a` | `editor::SelectAll` | `Editor` | bind | Ctrl+A replaces Zed's emacs-style line start |
| `EditorAddCaretPerSelectedLine` | Shift+Alt+G | `alt-shift-g` | `editor::SplitSelectionIntoLines` | `Editor` | bind |  |
| `EditorAddOrRemoveCaret` | mouse control alt button1 |  |  |  | zed | mouse: Alt+click adds a cursor, Shift+Alt+drag selects a column (multi_cursor_modifier) |
| `EditorAddRectangularSelectionOnMouseDrag` | mouse ctrl alt shift button1 |  |  |  | zed | mouse: Shift+Alt+drag |
| `EditorCodeBlockEndWithSelection` | Shift+Ctrl+Alt+] | `ctrl-alt-}` | `editor::SelectToEndOfLargerSyntaxNode` | `Editor && mode == full` | bind |  |
| `EditorCodeBlockStartWithSelection` | Shift+Ctrl+Alt+[ | `ctrl-alt-{` | `editor::SelectToStartOfLargerSyntaxNode` | `Editor && mode == full` | bind |  |
| `EditorCreateRectangularSelection` | mouse shift alt button1 |  |  |  | zed | mouse: Alt+click adds a cursor, Shift+Alt+drag selects a column (multi_cursor_modifier) |
| `EditorCreateRectangularSelectionOnMouseDrag` | mouse alt button1, shift alt button1 |  |  |  | zed | mouse: Alt+click adds a cursor, Shift+Alt+drag selects a column (multi_cursor_modifier) |
| `EditorLeftWithSelection` | Shift+Left | `shift-left` |  | `Editor` | zed | editor::SelectLeft, SelectRight |
| `EditorNextWordWithSelection` | Shift+Ctrl+Right | `ctrl-shift-right` | `editor::SelectToNextWordEnd` | `Editor` | bind | replaces Zed's SelectLargerSyntaxNode |
| `EditorPreviousWordWithSelection` | Shift+Ctrl+Left | `ctrl-shift-left` | `editor::SelectToPreviousWordStart` | `Editor` | bind | replaces Zed's SelectSmallerSyntaxNode |
| `EditorRightWithSelection` | Shift+Right | `shift-right` |  | `Editor` | zed | editor::SelectLeft, SelectRight |
| `EditorSelectWord` | Shift+Alt+=; Ctrl+W | `alt-+`; `ctrl-w` | `editor::SelectLargerSyntaxNode` | `Editor` | bind |  |
| `EditorUnSelectWord` | Shift+Alt+-; Shift+Ctrl+W | `alt-_`; `ctrl-shift-w` | `editor::SelectSmallerSyntaxNode` | `Editor` | bind |  |
| `ReSharperSelectContainingDeclaration` | Shift+Alt+]; Shift+Ctrl+[ | `alt-}`; `ctrl-{` | `editor::SelectEnclosingSymbol` | `Editor && mode == full` | bind |  |
| `SelectAllOccurrences` | Shift+Alt+; | `alt-:` | `editor::SelectAllMatches` | `Editor` | bind |  |
| `SelectNextOccurrence` | Shift+Alt+. | `alt->` | `["editor::SelectNext", {"replace_newest": false}]` | `Editor` | bind |  |
| `SmartSelect` | Shift+Alt+=; Ctrl+W | `alt-+`; `ctrl-w` | `editor::SelectLargerSyntaxNode` | `Editor` | bind |  |
| `UnselectPreviousOccurrence` | Shift+Alt+, | `alt-<` | `editor::UndoSelection` | `Editor` | approx | undoes the last selection change |

## Completion

| F12 action | Keys | Zed keys | Zed action | Context | Status | Note |
| --- | --- | --- | --- | --- | --- | --- |
| `ClassNameCompletion` | Shift+Alt+Space | `alt-shift-space` | `editor::ShowCompletions` | `Editor && mode == full` | approx | one kind of completion |
| `CodeCompletion` | Ctrl+Space; Ctrl+J | `ctrl-space`; `ctrl-j` | `editor::ShowCompletions` | `Editor && mode == full` | bind | Ctrl+Space is Zed's, but macOS's input sources may take it |
| `EditorChooseLookupItem` | Enter | `enter` |  | `Editor` | zed | Zed's completion and snippet keys |
| `EditorChooseLookupItemReplace` | Tab | `tab` |  | `Editor` | zed | Zed's completion and snippet keys |
| `EditorLookupDown` | Ctrl+Down | `ctrl-down` |  | `Editor` | zed | Zed's completion and snippet keys |
| `EditorLookupUp` | Ctrl+Up | `ctrl-up` |  | `Editor` | zed | Zed's completion and snippet keys |
| `ExpandLiveTemplateByTab` | Tab | `tab` |  | `Editor` | zed | Zed's completion and snippet keys |
| `HippieCompletion` | Alt+/ | `alt-/` | `editor::ShowWordCompletions` | `Editor && mode == full` | approx | words of the buffer; takes Alt+/ from which-key in the editor |
| `InsertNextEditAction` | Tab | `tab` | `editor::AcceptEditPrediction` | `Editor` | zed | Tab accepts Zed's edit prediction |
| `MethodOverloadSwitchDown` | Ctrl+Down | `ctrl-down` | `editor::SignatureHelpNext` | `Editor && showing_signature_help` | bind |  |
