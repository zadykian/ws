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
| `MethodOverloadSwitchUp` | Ctrl+Up | `ctrl-up` | `editor::SignatureHelpPrevious` | `Editor && showing_signature_help` | bind |  |
| `NextParameter` | Tab | `tab` |  | `Editor` | zed | Zed's completion and snippet keys |
| `NextTemplateVariable` | Tab; Enter | `tab`; `enter` |  | `Editor` | zed | Zed's completion and snippet keys |
| `ParameterInfo` | Shift+Ctrl+Space | `ctrl-shift-space` | `editor::ShowSignatureHelp` | `Editor && mode == full` | bind | replaces Zed's ShowWordCompletions |
| `ParameterInfo.GoToPreviousSignature` | Shift+Ctrl+P | `ctrl-shift-p` | `editor::SignatureHelpPrevious` | `Editor && showing_signature_help` | bind | Pause's key too, outside the popup |
| `PreviousTemplateVariable` | Shift+Tab | `shift-tab` |  | `Editor` | zed | Zed's completion and snippet keys |
| `PrevParameter` | Shift+Tab | `shift-tab` |  | `Editor` | zed | Zed's completion and snippet keys |
| `SmartTypeCompletion` | Ctrl+Alt+Space | `ctrl-alt-space` | `editor::ShowCompletions` | `Editor && mode == full` | approx | one kind of completion |

## Navigation

| F12 action | Keys | Zed keys | Zed action | Context | Status | Note |
| --- | --- | --- | --- | --- | --- | --- |
| `Back` | Ctrl+-; Ctrl+numpad - | `ctrl--` | `pane::GoBack` | `(none)` | zed | Ctrl+- is Zed's in a pane; bound globally for the panels |
| `CallHierarchy` | Ctrl+K, T; Ctrl+K, Ctrl+T | `ctrl-k t`; `ctrl-k ctrl-t` | `call_hierarchy::ShowIncomingCalls` | `Editor && mode == full` | bind |  |
| `CallHierarchy.BaseOnThisMethod` | Ctrl+K, T; Ctrl+K, Ctrl+T | `ctrl-k t`; `ctrl-k ctrl-t` | `call_hierarchy::ShowIncomingCalls` | `Editor && mode == full` | bind |  |
| `CopyReference` | Shift+Ctrl+Alt+C | `ctrl-alt-shift-c` | `editor::CopyFileLocation` | `Editor && mode == full` | approx | path:line, not the symbol's name |
| `EditorCodeBlockStart` | Shift+Alt+[ | `alt-{` | `editor::MoveToStartOfLargerSyntaxNode` | `Editor && mode == full` | bind |  |
| `EditorDown` | Down | `down` |  | `Editor` | zed | Zed's editor keys |
| `EditorDownWithSelection` | Shift+Down | `shift-down` |  | `Editor` | zed | Zed's editor keys |
| `EditorLeft` | Left | `left` |  | `Editor` | zed | Zed's editor keys |
| `EditorLineEnd` | End | `end` |  | `Editor` | zed | Zed's editor keys |
| `EditorLineEndWithSelection` | Shift+End | `shift-end` |  | `Editor` | zed | Zed's editor keys |
| `EditorLineStart` | Home | `home` |  | `Editor` | zed | Zed's editor keys |
| `EditorLineStartWithSelection` | Shift+Home | `shift-home` |  | `Editor` | zed | Zed's editor keys |
| `EditorMatchBrace` | Ctrl+] | `ctrl-]` | `editor::MoveToEnclosingBracket` | `Editor && mode == full` | bind | in the full editor, so the inline assistant keeps Ctrl+] for its next alternative |
| `EditorNextWord` | Ctrl+Right | `ctrl-right` | `editor::MoveToNextWordEnd` | `Editor` | bind | macOS's Spaces take Ctrl+Right by default |
| `EditorPageDown` | PgDn | `pagedown` |  | `Editor` | zed | Zed's editor keys |
| `EditorPageDownWithSelection` | Shift+PgDn | `shift-pagedown` |  | `Editor` | zed | Zed's editor keys |
| `EditorPageUp` | PgUp | `pageup` |  | `Editor` | zed | Zed's editor keys |
| `EditorPageUpWithSelection` | Shift+PgUp | `shift-pageup` |  | `Editor` | zed | Zed's editor keys |
| `EditorPreviousWord` | Ctrl+Left | `ctrl-left` | `editor::MoveToPreviousWordStart` | `Editor` | bind | macOS's Spaces take Ctrl+Left by default |
| `EditorRight` | Right | `right` |  | `Editor` | zed | Zed's editor keys |
| `EditorScrollDown` | Ctrl+Down | `ctrl-down` | `editor::LineDown` | `Editor` | bind | App windows takes Ctrl+Down by default |
| `EditorScrollToCenter` | Ctrl+M, C; Ctrl+M, Ctrl+C | `ctrl-m c`; `ctrl-m ctrl-c` | `editor::ScrollCursorCenter` | `Editor` | bind |  |
| `EditorScrollUp` | Ctrl+Up | `ctrl-up` | `editor::LineUp` | `Editor` | bind | Mission Control takes Ctrl+Up by default |
| `EditorUp` | Up | `up` |  | `Editor` | zed | Zed's editor keys |
| `EditorUpWithSelection` | Shift+Up | `shift-up` |  | `Editor` | zed | Zed's editor keys |
| `ExternalJavaDoc` | Shift+F1 | `shift-f1` | `editor::OpenDocs` | `Editor && mode == full` | approx | where the server gives a docs link |
| `FindUsages` | Shift+F12 | `shift-f12` | `editor::FindAllReferences` | `Editor && mode == full` | bind | replaces Zed's Shift+F12, GoToImplementation |
| `Forward` | Ctrl+=; mouse button5 | `ctrl-=` | `pane::GoForward` | `(none)` | bind | mouse button 5 is Zed's already |
| `GoCallHierarchyPopupMenu.BaseOnThisDeclaration` | Ctrl+K, T; Ctrl+K, Ctrl+T | `ctrl-k t`; `ctrl-k ctrl-t` | `call_hierarchy::ShowIncomingCalls` | `Editor && mode == full` | bind |  |
| `GotoDeclaration` | Alt+D | `alt-d` | `editor::GoToDefinition` | `Editor && mode == full` | bind | F12 stays Zed's too |
| `GotoImplementation` | Ctrl+Alt+D | `ctrl-alt-d` | `editor::GoToImplementation` | `Editor && mode == full` | bind |  |
| `GotoNextBookmark` | Ctrl+K, N; Ctrl+K, Ctrl+N | `ctrl-k n`; `ctrl-k ctrl-n` | `editor::GoToNextBookmark` | `Editor && mode == full` | bind |  |
| `GotoNextElementUnderCaretUsage` | Shift+Ctrl+Down | `ctrl-shift-down` | `editor::GoToNextDocumentHighlight` | `Editor && mode == full` | bind | Find next's Shift+Ctrl+Down goes here |
| `GotoPrevElementUnderCaretUsage` | Shift+Ctrl+Up | `ctrl-shift-up` | `editor::GoToPreviousDocumentHighlight` | `Editor && mode == full` | bind |  |
| `GotoPreviousBookmark` | Ctrl+K, P; Ctrl+K, Ctrl+P | `ctrl-k p`; `ctrl-k ctrl-p` | `editor::GoToPreviousBookmark` | `Editor && mode == full` | bind |  |
| `GotoSuperMethod` | Alt+Home | `alt-home` | `editor::GoToImplementation` | `Editor && mode == full` | approx | gopls lists a method's interfaces too; C# goes down alone |
| `GotoTypeDeclaration` | Shift+Ctrl+F11 | `ctrl-shift-f11` | `editor::GoToTypeDefinition` | `Editor && mode == full` | bind |  |
| `JumpToLastChange` | Shift+Ctrl+Backspace | `ctrl-shift-backspace` | `editor::GoToPreviousChange` | `Editor && mode == full` | approx | the file's own changes, not across files |
| `QuickImplementations` | Alt+F12 | `alt-f12` | `editor::GoToDefinitionSplit` | `Editor && mode == full` | zed | no popup: Zed's Alt+F12 opens it in a split |
| `QuickJavaDoc` | Ctrl+K, I; Ctrl+K, Ctrl+I; Shift+Ctrl+F1 | `ctrl-k i`; `ctrl-k ctrl-i`; `ctrl-shift-f1` | `editor::Hover` | `Editor && mode == full` | bind |  |
| `ReSharperGotoContainingDeclaration` | Ctrl+[ | `ctrl-[` | `["action::Sequence", ["editor::SelectEnclosingSymbol", "editor::MoveLeft"]]` | `Editor && mode == full` | approx | selects the symbol, then collapses to its start |
| `ShowSettingsAndFindUsages` | Shift+Ctrl+Alt+F12 | `ctrl-alt-shift-f12` | `editor::FindAllReferences` | `Editor && mode == full` | approx | no settings dialog |
| `ShowUsages` | Shift+Alt+F12 | `alt-shift-f12` | `editor::FindAllReferences` | `Editor && mode == full` | zed | Shift+Alt+F12 is Zed's; a multibuffer, not a popup |
| `ToggleBookmarkWithMnemonic` | Ctrl+F11 | `ctrl-f11` | `editor::ToggleBookmarkWithLabel` | `Editor && mode == full` | approx | a label, not a mnemonic key |

## Search

| F12 action | Keys | Zed keys | Zed action | Context | Status | Note |
| --- | --- | --- | --- | --- | --- | --- |
| `FileStructurePopup` | Alt+\ | `alt-\` | `outline::Toggle` | `Editor && mode == full` | bind |  |
| `Find` | Cmd+F; Ctrl+F | `cmd-f`; `ctrl-f` | `buffer_search::Deploy` | `(none)` | bind | Cmd+F is Zed's already |
| `FindInPath` | Shift+Cmd+F; Shift+Ctrl+F | `cmd-shift-f`; `ctrl-shift-f` | `pane::DeploySearch` | `(none)` | bind | Shift+Cmd+F is Zed's already |
| `FindNext` | Shift+Ctrl+Down; Shift+Alt+/ | (not `ctrl-shift-down`); `alt-?` | `search::SelectNextMatch` | `(none)` | approx | Shift+Alt+/ alone: Shift+Ctrl+Down goes to the next highlighted usage |
| `FindPrevWordAtCaret` | Shift+Ctrl+F3 | `ctrl-shift-f3` | `editor::FindPreviousMatch` | `Editor` | approx |  |
| `FindWordAtCaret` | Ctrl+F3 | `ctrl-f3` | `editor::FindNextMatch` | `Editor` | approx | Ctrl+F3: F-keys need Fn or the setting |
| `GotoAction` | Ctrl+Q | `ctrl-q` | `command_palette::Toggle` | `(none)` | bind |  |
| `GotoFile` | Shift+Ctrl+T | `ctrl-shift-t` | `file_finder::Toggle` | `(none)` | bind |  |
| `GotoSymbol` | Shift+Ctrl+Alt+T | `ctrl-alt-shift-t` | `project_symbols::Toggle` | `(none)` | bind |  |
| `OpenFile` | Ctrl+O | `ctrl-o` | `workspace::Open` | `(none)` | approx | opens a folder; on a remote project, the server's |
| `Replace` | Ctrl+H | `ctrl-h` | `buffer_search::DeployReplace` | `(none)` | bind | replaces Zed's emacs-style backspace |
| `ReplaceInPath` | Shift+Ctrl+H | `ctrl-shift-h` | `["pane::DeploySearch", {"replace_enabled": true}]` | `(none)` | bind |  |
| `SearchEverywhere` | Ctrl+, | `ctrl-,` | `file_finder::Toggle` | `(none)` | approx | files alone, as in helix |
| `TextSearchAction` | Shift+Ctrl+Alt+E | `ctrl-alt-shift-e` | `text_finder::Toggle` | `(none)` | bind |  |

## Code

| F12 action | Keys | Zed keys | Zed action | Context | Status | Note |
| --- | --- | --- | --- | --- | --- | --- |
| `AIAssistant.Editor.AskAiAssistantInEditor` | Ctrl+/ | `ctrl-/` | `assistant::InlineAssist` | `Editor && mode == full` | bind | the inline assistant, whose Ctrl+Enter is F12's code actions here |
| `ChangeSignature` | Ctrl+R, S | `ctrl-r s` | `editor::ToggleCodeActions` | `Editor && mode == full` | approx | the server's code actions, of every kind |
| `ExtractMethod` | Ctrl+R, M | `ctrl-r m` | `editor::ToggleCodeActions` | `Editor && mode == full` | approx | the server's code actions, of every kind |
| `Generate` | Alt+Insert | (no Insert) | `editor::ToggleCodeActions` | `Editor && mode == full` | approx | Alt+Insert: a Mac keyboard has no Insert |
| `GoFmtFileAction` | Shift+Ctrl+Alt+F | `ctrl-alt-shift-f` | `editor::Format` | `Editor && mode == full` | bind |  |
| `ImplementMethods` | Shift+Ctrl+I | `ctrl-shift-i` | `editor::ToggleCodeActions` | `Editor && mode == full` | approx | the server's code actions, of every kind |
| `IntroduceConstant` | Ctrl+Alt+C | `ctrl-alt-c` | `editor::ToggleCodeActions` | `Editor && mode == full` | approx | the server's code actions, of every kind |
| `Refactorings.QuickListPopupAction` | Ctrl+T | `ctrl-t` | `editor::ToggleCodeActions` | `Editor && mode == full` | approx | the server's code actions, of every kind |
| `ReformatCode` | Ctrl+Alt+Enter; Ctrl+K, F; Ctrl+K, Ctrl+F; Ctrl+K, D; Ctrl+K, Ctrl+D | `ctrl-alt-enter`; `ctrl-k f`; `ctrl-k ctrl-f`; `ctrl-k d`; `ctrl-k ctrl-d` | `editor::Format` | `Editor && mode == full` | bind |  |
| `RenameElement` | Ctrl+R, R | `ctrl-r r` | `editor::Rename` | `Editor && mode == full` | bind | Zed's F2 stays too |
| `ShowErrorDescription` | Ctrl+E | `ctrl-e` | `editor::Hover` | `Editor && mode == full` | approx | the hover shows the diagnostic |
| `ShowIntentionActions` | Alt+Enter; Shift+Enter; Ctrl+Enter | `alt-enter`; `shift-enter`; `ctrl-enter` | `editor::ToggleCodeActions` | `Editor && mode == full` | bind | Shift+Enter and Ctrl+Enter only in the full editor |
| `ShowReformatFileDialog` | Shift+Ctrl+Alt+L | `ctrl-alt-shift-l` | `editor::Format` | `Editor && mode == full` | bind |  |
| `ToggleInlayHintsGloballyAction` | Ctrl+I | `ctrl-i` | `editor::ToggleInlayHints` | `Editor` | bind |  |

## Folding

| F12 action | Keys | Zed keys | Zed action | Context | Status | Note |
| --- | --- | --- | --- | --- | --- | --- |
| `CollapseAll` | Ctrl+numpad -; Ctrl+- | `ctrl--` | `project_panel::CollapseAllEntries` | `ProjectPanel && not_editing` | bind | trees: the project panel |
| `CollapseAllRegions` | Ctrl+M, A; Ctrl+M, Ctrl+A | `ctrl-m a`; `ctrl-m ctrl-a` | `editor::FoldAll` | `Editor && mode == full` | bind |  |
| `CollapseRegion` | Ctrl+M, S; Ctrl+M, Ctrl+S | `ctrl-m s`; `ctrl-m ctrl-s` | `editor::Fold` | `Editor && mode == full` | bind |  |
| `CollapseRegionRecursively` | Ctrl+Alt+numpad -; Ctrl+Alt+- | `ctrl-alt--` | `editor::FoldRecursive` | `Editor && mode == full` | bind | Ctrl+Alt+- in the editor, not the dock's size |
| `CollapseSelection` | Ctrl+M, H; Ctrl+M, Ctrl+H; Ctrl+M, U; Ctrl+M, Ctrl+U | `ctrl-m h`; `ctrl-m ctrl-h`; `ctrl-m u`; `ctrl-m ctrl-u` | `editor::FoldSelectedRanges` | `Editor && mode == full` | bind |  |
| `ExpandAll` | Ctrl+numpad + | `ctrl-+` | `project_panel::ExpandAllEntries` | `ProjectPanel && not_editing` | approx | Ctrl+numpad +, which a Mac sees as Ctrl++ |
| `ExpandAllRegions` | Ctrl+M, X; Ctrl+M, Ctrl+X | `ctrl-m x`; `ctrl-m ctrl-x` | `editor::UnfoldAll` | `Editor && mode == full` | bind |  |
| `ExpandAllToLevel1` | Shift+Ctrl+`numpad *`, 1; Shift+Ctrl+`numpad *`, numpad 1 | `ctrl-* 1` | `editor::FoldAtLevel_1` | `Editor && mode == full` | approx | folds below the level; both chords are Ctrl+* on a Mac |
| `ExpandAllToLevel2` | Shift+Ctrl+`numpad *`, 2; Shift+Ctrl+`numpad *`, numpad 2 | `ctrl-* 2` | `editor::FoldAtLevel_2` | `Editor && mode == full` | approx | folds below the level; both chords are Ctrl+* on a Mac |
| `ExpandAllToLevel3` | Shift+Ctrl+`numpad *`, 3; Shift+Ctrl+`numpad *`, numpad 3 | `ctrl-* 3` | `editor::FoldAtLevel_3` | `Editor && mode == full` | approx | folds below the level; both chords are Ctrl+* on a Mac |
| `ExpandAllToLevel4` | Shift+Ctrl+`numpad *`, 4; Shift+Ctrl+`numpad *`, numpad 4 | `ctrl-* 4` | `editor::FoldAtLevel_4` | `Editor && mode == full` | approx | folds below the level; both chords are Ctrl+* on a Mac |
| `ExpandAllToLevel5` | Shift+Ctrl+`numpad *`, 5; Shift+Ctrl+`numpad *`, numpad 5 | `ctrl-* 5` | `editor::FoldAtLevel_5` | `Editor && mode == full` | approx | folds below the level; both chords are Ctrl+* on a Mac |
| `ExpandCollapseToggleAction` | Ctrl+M, M; Ctrl+M, Ctrl+M | `ctrl-m m`; `ctrl-m ctrl-m` | `editor::ToggleFold` | `Editor && mode == full` | bind |  |
| `ExpandRegion` | Ctrl+M, E; Ctrl+M, Ctrl+E | `ctrl-m e`; `ctrl-m ctrl-e` | `editor::UnfoldLines` | `Editor && mode == full` | bind |  |
| `ExpandRegionRecursively` | Ctrl+Alt+numpad +; Ctrl+Alt+= | `ctrl-alt-+`; `ctrl-alt-=` | `editor::UnfoldRecursive` | `Editor && mode == full` | bind |  |
| `ExpandToLevel1` | Ctrl+`numpad *`, 1; Ctrl+`numpad *`, numpad 1 | `ctrl-* 1` | `editor::FoldAtLevel_1` | `Editor && mode == full` | approx | folds below the level; both chords are Ctrl+* on a Mac |
| `ExpandToLevel2` | Ctrl+`numpad *`, 2; Ctrl+`numpad *`, numpad 2 | `ctrl-* 2` | `editor::FoldAtLevel_2` | `Editor && mode == full` | approx | folds below the level; both chords are Ctrl+* on a Mac |
| `ExpandToLevel3` | Ctrl+`numpad *`, 3; Ctrl+`numpad *`, numpad 3 | `ctrl-* 3` | `editor::FoldAtLevel_3` | `Editor && mode == full` | approx | folds below the level; both chords are Ctrl+* on a Mac |
| `ExpandToLevel4` | Ctrl+`numpad *`, 4; Ctrl+`numpad *`, numpad 4 | `ctrl-* 4` | `editor::FoldAtLevel_4` | `Editor && mode == full` | approx | folds below the level; both chords are Ctrl+* on a Mac |
| `ExpandToLevel5` | Ctrl+`numpad *`, 5; Ctrl+`numpad *`, numpad 5 | `ctrl-* 5` | `editor::FoldAtLevel_5` | `Editor && mode == full` | approx | folds below the level; both chords are Ctrl+* on a Mac |
| `ExternalSystem.CollapseAll` | Ctrl+numpad -; Ctrl+- | `ctrl--` | `project_panel::CollapseAllEntries` | `ProjectPanel && not_editing` | bind | trees: the project panel |
| `ExternalSystem.ExpandAll` | Ctrl+numpad + | `ctrl-+` | `project_panel::ExpandAllEntries` | `ProjectPanel && not_editing` | approx | Ctrl+numpad +, which a Mac sees as Ctrl++ |
| `RiderCollapseToDefinitions` | Ctrl+M, O; Ctrl+M, Ctrl+O | `ctrl-m o`; `ctrl-m ctrl-o` | `editor::FoldFunctionBodies` | `Editor && mode == full` | approx |  |
