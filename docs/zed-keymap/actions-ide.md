# The actions the Zed keymap binds, in the IDE

Each row is an F12 action with keys, and what Zed does with them. The status `bind` is the same
behaviour, `approx` the closest Zed action, and `zed` that Zed's own default already does it.
F12's actions come with those of IntelliJ's default keymap that F12 leaves alone. This page covers
Problems to Terminal, and three actions of the default keymap's Other.

## Problems

| F12 action | Keys | Zed keys | Zed action | Context | Status | Note |
| --- | --- | --- | --- | --- | --- | --- |
| `ActivateErrorsInSolutionToolWindow` | Ctrl+Alt+2 | `ctrl-alt-2` | `diagnostics::Deploy` | `(none)` | bind |  |
| `ActivateInspectionResultsToolWindow` | Ctrl+Alt+4; Ctrl+Alt+V | `ctrl-alt-4`; `ctrl-alt-v` | `diagnostics::Deploy` | `(none)` | bind |  |
| `CodeInspection.OnEditor` | Shift+Alt+I | `alt-shift-i` | `diagnostics::DeployCurrentFile` | `(none)` | bind |  |
| `GotoNextError` | Alt+PgDn | `alt-pagedown` | `editor::GoToDiagnostic` | `Editor && mode == full` | bind |  |
| `GotoPreviousError` | Alt+PgUp | `alt-pageup` | `editor::GoToPreviousDiagnostic` | `Editor && mode == full` | bind |  |
| `InspectCode` | Alt+F11 | `alt-f11` | `diagnostics::Deploy` | `(none)` | bind |  |
| `ReSharperGotoNextErrorInSolution` | Shift+Alt+PgDn; Shift+Ctrl+F12 | `alt-shift-pagedown`; `ctrl-shift-f12` | `diagnostics::Deploy` | `(none)` | approx | opens the project's diagnostics, where F12's Alt+PgDn moves on |
| `ReSharperGotoPrevErrorInSolution` | Shift+Alt+F2; Shift+Alt+PgUp | `alt-shift-f2`; `alt-shift-pageup` | `diagnostics::Deploy` | `(none)` | approx | opens the project's diagnostics, where F12's Alt+PgDn moves on |
| `SwitcherNextProblemFallback` | Alt+PgDn | `alt-pagedown` | `editor::GoToDiagnostic` | `Editor && mode == full` | bind |  |

## Files, tabs and splits

| F12 action | Keys | Zed keys | Zed action | Context | Status | Note |
| --- | --- | --- | --- | --- | --- | --- |
| `CloseActiveTab` | Shift+Ctrl+F4 | `ctrl-shift-f4` | `workspace::CloseActiveDock` | `(none)` | approx |  |
| `CloseContent` | Ctrl+F4 | `ctrl-f4` | `pane::CloseActiveItem` | `(none)` | bind | macOS's keyboard navigation may take Ctrl+F4 |
| `DownloadBackendFileToClient` | Shift+Ctrl+D | `ctrl-shift-d` | `project_panel::DownloadFromRemote` | `ProjectPanel && not_editing` | approx | the project panel's file, from the server |
| `FocusEditor` | Esc | `escape` | `editor::ToggleFocus` | `Dock \|\| OutlinePanel \|\| ProjectPanel \|\| GitPanel` | bind | Esc from a panel; the terminal keeps Esc |
| `HideAllWindows` | Shift+Esc | `shift-escape` | `workspace::ToggleAllDocks` | `(none)` | bind | replaces Zed's Shift+Esc, ToggleZoom |
| `LocateInSolutionView` | Shift+Alt+L | `alt-shift-l` | `pane::RevealInProjectPanel` | `(none)` | bind |  |
| `MaximizeToolWindow` | Shift+Ctrl+' | `ctrl-"` | `workspace::ToggleZoom` | `(none)` | bind |  |
| `NewElement` | Ctrl+N; Alt+Insert | `ctrl-n`; (no Insert) | `workspace::NewFile` | `Editor && mode == full` | approx | an untitled buffer; in the project panel (`ProjectPanel && not_editing`), `project_panel::NewFile`, a file there |
| `NewElementSamePlace` | Shift+Ctrl+A | `ctrl-shift-a` | `project_panel::NewFile` | `ProjectPanel && not_editing` | bind |  |
| `NextProjectWindow` | Ctrl+Alt+] | `ctrl-alt-]` | `workspace::ActivateNextWindow` | `(none)` | bind |  |
| `NextSplitter` | Ctrl+End; Shift+Alt+Right | `ctrl-end`; `alt-shift-right` | `workspace::ActivateNextPane` | `(none)` | bind | takes Shift+Alt+Right, macOS's select word |
| `NextTab` | Ctrl+Alt+PgDn; Alt+Right | `ctrl-alt-pagedown`; `alt-right` | `pane::ActivateNextItem` | `(none)` | bind | takes Alt+Right from Zed's word motion |
| `PreviousProjectWindow` | Ctrl+Alt+[ | `ctrl-alt-[` | `workspace::ActivatePreviousWindow` | `(none)` | bind |  |
| `PreviousTab` | Ctrl+Alt+PgUp; Alt+Left | `ctrl-alt-pageup`; `alt-left` | `pane::ActivatePreviousItem` | `(none)` | bind | takes Alt+Left from Zed's word motion |
| `PrevSplitter` | Ctrl+Home; Shift+Alt+Left | `ctrl-home`; `alt-shift-left` | `workspace::ActivatePreviousPane` | `(none)` | bind |  |
| `ResizeToolWindowDown` | Ctrl+Alt+Down | `ctrl-alt-down` | `workspace::DecreaseActiveDockSize` | `(none)` | approx | by the dock, whichever its side |
| `ResizeToolWindowLeft` | Ctrl+Alt+Left | `ctrl-alt-left` | `workspace::DecreaseActiveDockSize` | `(none)` | approx | by the dock, whichever its side |
| `ResizeToolWindowRight` | Ctrl+Alt+Right | `ctrl-alt-right` | `workspace::IncreaseActiveDockSize` | `(none)` | approx |  |
| `ResizeToolWindowUp` | Ctrl+Alt+Up | `ctrl-alt-up` | `workspace::IncreaseActiveDockSize` | `(none)` | approx |  |
| `SelectIn` | Alt+F1 | `alt-f1` | `pane::RevealInProjectPanel` | `(none)` | approx |  |
| `SplitVertically` | Shift+Ctrl+Alt+R | `ctrl-alt-shift-r` | `pane::SplitRight` | `(none)` | bind |  |
| `Switcher` | Shift+Ctrl+Tab | `ctrl-shift-tab` | `["tab_switcher::Toggle", {"select_last": true}]` | `(none)` | zed | Shift+Ctrl+Tab is Zed's |
| `ToggleFullScreen` | Shift+Alt+Enter | `alt-shift-enter` | `zed::ToggleFullScreen` | `(none)` | bind | Ctrl+Cmd+F stays Zed's too |

## Tool windows

| F12 action | Keys | Zed keys | Zed action | Context | Status | Note |
| --- | --- | --- | --- | --- | --- | --- |
| `ActivateAIAssistantToolWindow` | F7; Alt+7 | `f7`; `alt-7` | `agent::ToggleFocus` | `(none)` | bind | the agent panel; replaces Zed's F7, StepOver while stopped, as F10 steps over too |
| `ActivateCommitToolWindow` | F5; Alt+5 | `f5`; `alt-5` | `git_panel::ToggleFocus` | `(none)` | bind | replaces Zed's F5, debugger::Rerun |
| `ActivateProjectToolWindow` | F1; Alt+1 | `f1`; `alt-1` | `project_panel::ToggleFocus` | `(none)` | bind |  |
| `ActivateTerminalToolWindow` | F2; Alt+2 | `f2`; `alt-2` | `terminal_panel::Toggle` | `(none)` | bind | in the terminal, it closes the dock |
| `ActivateVersionControlToolWindow` | F6; Alt+6 | `f6`; `alt-6` | `git_graph::Open` | `(none)` | approx | the log: Zed's git graph |
| `HelpTopics` | Ctrl+F1 | `ctrl-f1` | `zed::OpenDocs` | `(none)` | bind |  |
| `ShowPopupMenu` | Menu; F13 | (no the menu key); `f13` | `editor::OpenContextMenu` | `Editor && mode == full` | bind | F13 alone: a Mac has no menu key |

## Version control

| F12 action | Keys | Zed keys | Zed action | Context | Status | Note |
| --- | --- | --- | --- | --- | --- | --- |
| `ChangesView.AddUnversioned` | Ctrl+Alt+A | `ctrl-alt-a` | `git::ToggleStaged` | `GitPanel` | approx | in the git panel |
| `ChangesView.GroupBy.Directory` | Ctrl+Alt+P | `ctrl-alt-p` | `git_panel::ToggleTreeView` | `GitPanel` | approx |  |
| `ChangesView.Revert` | Ctrl+Alt+Z | `ctrl-alt-z` | `git::RestoreFile` | `GitPanel` | approx |  |
| `ChangesView.ShelveSilently` | Shift+Ctrl+H | `ctrl-shift-h` | `git::StashAll` | `GitPanel` | approx | a stash, not a shelf; in the git panel, as Shift+Ctrl+H is Replace in path's |
| `ChangesView.ToggleCommitUi` | Ctrl+Alt+K | `ctrl-alt-k` | `git::ToggleCommitEditor` | `(none)` | approx |  |
| `ChangesView.UnshelveSilently` | Ctrl+Alt+U | `ctrl-alt-u` | `git::StashPop` | `GitPanel` | approx |  |
| `CheckinProject` | Ctrl+Alt+K | `ctrl-alt-k` | `git::ToggleCommitEditor` | `(none)` | approx |  |
| `Git.Add` | Ctrl+Alt+A | `ctrl-alt-a` | `git::ToggleStaged` | `GitPanel` | approx | in the git panel |
| `Git.Branches` | Alt+X | `alt-x` | `git::Branch` | `(none)` | bind |  |
| `Git.Commit.And.Push.Executor` | Alt+C | `alt-c` | `git::Commit` | `CommitEditor > Editor` | approx | commits; push stays separate |
| `Git.Commit.Stage` | Ctrl+Alt+K | `ctrl-alt-k` | `git::ToggleCommitEditor` | `(none)` | approx |  |
| `Git.CreateNewBranch` | Ctrl+Alt+N | `ctrl-alt-n` | `git::Branch` | `(none)` | approx | the branch picker creates one |
| `Git.CreateNewBranch.FromCommit` | Ctrl+Alt+N | `ctrl-alt-n` | `git::Branch` | `(none)` | approx | the branch picker creates one |
| `Git.Fetch` | Alt+Z | `alt-z` | `git::Fetch` | `(none)` | bind |  |
| `Git.New.Branch.In.Log` | Ctrl+Alt+N | `ctrl-alt-n` | `git::Branch` | `(none)` | approx | the branch picker creates one |
| `Git.ShowBranches` | Alt+X | `alt-x` | `git::Branch` | `(none)` | bind |  |
| `Git.Stage.Add` | Ctrl+Alt+A | `ctrl-alt-a` | `git::ToggleStaged` | `GitPanel` | approx | in the git panel |
| `Git.Stage.Revert` | Ctrl+Alt+Z | `ctrl-alt-z` | `git::RestoreFile` | `GitPanel` | approx |  |
| `hg4idea.add` | Ctrl+Alt+A | `ctrl-alt-a` | `git::ToggleStaged` | `GitPanel` | approx | in the git panel |
| `NextDiff` | F8 | `f8` | `editor::GoToHunk` | `Editor && mode == full` | bind | replaces Zed's F8, GoToDiagnostic |
| `PreviousDiff` | Shift+F8; Shift+F7 | `shift-f8`; `shift-f7` | `editor::GoToPreviousHunk` | `Editor && mode == full` | bind | replaces Zed's Shift+F8 |
| `Synchronize` | Ctrl+Alt+Y | `ctrl-alt-y` | `workspace::ReloadActiveItem` | `(none)` | approx | the active file |
| `Vcs.CopyRevisionNumberAction` | Shift+Ctrl+Alt+C | `ctrl-alt-shift-c` | `git_graph::CopyCommitSha` | `(none)` | approx | in the git graph |
| `Vcs.Push` | Alt+V | `alt-v` | `git::Push` | `(none)` | bind |  |
| `Vcs.RollbackChangedLines` | Ctrl+Alt+Z | `ctrl-alt-z` | `git::Restore` | `Editor && mode == full` | bind | the hunks at the cursor |
| `Vcs.ToggleAmendCommitMode` | Alt+A | `alt-a` | `git::Amend` | `CommitEditor > Editor` | approx |  |
| `VcsShowNextChangeMarker` | Shift+Ctrl+Alt+N | `ctrl-alt-shift-n` | `editor::GoToHunk` | `Editor && mode == full` | bind |  |
| `VcsShowPrevChangeMarker` | Shift+Ctrl+Alt+P | `ctrl-alt-shift-p` | `editor::GoToPreviousHunk` | `Editor && mode == full` | bind |  |

## Run and debug

| F12 action | Keys | Zed keys | Zed action | Context | Status | Note |
| --- | --- | --- | --- | --- | --- | --- |
| `BuildWholeSolutionAction` | Shift+Ctrl+B | `ctrl-shift-b` | `["task::Spawn", {"task_name": "build"}]` | `(none)` | approx | a task named build, which tasks.json gives |
| `ChooseDebugConfiguration` | Shift+Alt+F9 | `alt-shift-f9` | `debugger::Start` | `(none)` | bind | the new session's modal |
| `Debug` | Ctrl+F5 | `ctrl-f5` | `debugger::Rerun` | `(none)` | approx | the last debug session; Ctrl+F5 is macOS's toolbar key |
| `EditBreakpoint` | Alt+F9; Ctrl+Alt+B | `alt-f9`; `ctrl-alt-b` | `editor::EditLogBreakpoint` | `Editor && mode == full` | approx | a log message; conditions from the gutter's menu |
| `EvaluateExpression` | Alt+F | `alt-f` | `debugger::EvaluateSelectedText` | `Workspace && debugger_stopped` | approx | the selection, in the console |
| `ForceRunToCursor` | Ctrl+Alt+R | `ctrl-alt-r` | `debugger::RunToCursor` | `Workspace && debugger_stopped` | bind |  |
| `ForceStepInto` | Ctrl+Alt+E | `ctrl-alt-e` | `debugger::StepInto` | `Workspace && debugger_stopped` | bind |  |
| `ForceStepOver` | Ctrl+Alt+W | `ctrl-alt-w` | `debugger::StepOver` | `Workspace && debugger_stopped` | bind | while stopped, so Alt keys type otherwise |
| `GoGenerateFileAction` | Ctrl+Alt+G | `ctrl-alt-g` | `["task::Spawn", {"task_name": "go generate"}]` | `(none)` | approx | a task of that name |
| `Pause` | Ctrl+Alt+Break; Ctrl+Pause; Shift+Ctrl+P | (no Break); (no Pause); `ctrl-shift-p` | `debugger::Pause` | `(none)` | bind | Shift+Ctrl+P alone: a Mac has no Pause; global, as the editor's Shift+Ctrl+P would win |
| `QuickEvaluateExpression` | Ctrl+Alt+F8; mouse alt shift button1 | `ctrl-alt-f8` | `debugger::EvaluateSelectedText` | `Workspace && debugger_stopped` | approx |  |
| `Resume` | Shift+Ctrl+F5 | `ctrl-shift-f5` | `debugger::Continue` | `(none)` | bind |  |
| `RiderRemoveAllLineBreakpoints` | Shift+Ctrl+F9 | `ctrl-shift-f9` | `debugger::ClearAllBreakpoints` | `(none)` | bind |  |
| `RiderUnitTestRepeatPreviousRunAction` | Ctrl+U, U; Ctrl+U, Ctrl+U | `ctrl-u u`; `ctrl-u ctrl-u` | `task::Rerun` | `Editor && mode == full` | approx | in the editor alone, as a Ctrl+U chord would hold the shell's Ctrl+U |
| `RiderUnitTestRunContextAction` | Ctrl+U, R; Ctrl+U, Ctrl+R | `ctrl-u r`; `ctrl-u ctrl-r` | `editor::SpawnNearestTask` | `Editor && mode == full` | approx | the test at the cursor, as a task |
| `RiderUnitTestRunCurrentSessionAction` | Ctrl+U, Y; Ctrl+U, Ctrl+Y | `ctrl-u y`; `ctrl-u ctrl-y` | `task::Rerun` | `Editor && mode == full` | approx | in the editor alone, as a Ctrl+U chord would hold the shell's Ctrl+U |
| `Run` | Shift+F5 | `shift-f5` | `task::Rerun` | `(none)` | approx | the last task |
| `RunToCursor` | Alt+R | `alt-r` | `debugger::RunToCursor` | `Workspace && debugger_stopped` | bind |  |
| `StepInto` | Alt+E | `alt-e` | `debugger::StepInto` | `Workspace && debugger_stopped` | bind |  |
| `StepOut` | Alt+Q | `alt-q` | `debugger::StepOut` | `Workspace && debugger_stopped` | bind |  |
| `StepOver` | Alt+W | `alt-w` | `debugger::StepOver` | `Workspace && debugger_stopped` | bind | while stopped, so Alt keys type otherwise |
| `Stop` | Alt+F5 | `alt-f5` | `debugger::Stop` | `(none)` | bind |  |
| `ToggleBreakpointEnabled` | Ctrl+F9 | `ctrl-f9` | `debugger::ToggleEnableBreakpoint` | `Editor && mode == full` | bind |  |
| `ToggleLineBreakpoint` | Ctrl+B | `ctrl-b` | `editor::ToggleBreakpoint` | `Editor && mode == full` | bind | Ctrl+B in the editor alone: tmux's prefix in the terminal |
| `XDebugger.AttachToProcess` | Ctrl+Alt+P | `ctrl-alt-p` | `debugger::Start` | `(none)` | approx | the modal, on its Attach tab by hand |
| `XDebugger.RemoveWatch` | Delete | `delete` | `variable_list::RemoveWatch` | `VariableList` | zed | Delete is Zed's |

## Terminal

| F12 action | Keys | Zed keys | Zed action | Context | Status | Note |
| --- | --- | --- | --- | --- | --- | --- |
| `SendEOF` | Ctrl+D | `ctrl-d` |  | `Terminal` | zed | Ctrl+D reaches the shell |
| `Terminal.CopySelectedText` | Ctrl+C; Ctrl+Insert; Cmd+C | `ctrl-c`; (no Insert); `cmd-c` | `terminal::Copy` | `Terminal` | zed | Cmd+C is Zed's; Ctrl+C stays the shell's; a Mac has no Insert |
| `Terminal.Paste` | Ctrl+V; Shift+Insert; Cmd+V | `ctrl-v`; (no Insert); `cmd-v` | `terminal::Paste` | `Terminal` | bind |  |

## Other

| F12 action | Keys | Zed keys | Zed action | Context | Status | Note |
| --- | --- | --- | --- | --- | --- | --- |
| `ResetIdeScaleAction` | Shift+Alt+0 | `alt-)` | `zed::ResetUiFontSize` | `(none)` | bind |  |
| `ShowProjectStructureSettings` | Shift+Ctrl+Alt+S | `ctrl-alt-shift-s` | `zed::OpenProjectSettings` | `(none)` | approx |  |
| `ShowSettings` | Ctrl+Alt+S | `ctrl-alt-s` | `zed::OpenSettings` | `(none)` | bind |  |
