# The actions the Zed keymap lacks, L to Z

Each row is an F12 action that `mac/zed-keymap.json` binds no key for. The status `none` means
Zed lacks the action. The status `skip` means Zed has it, but the binding is left out. The
status `n/a` means a tool Zed lacks, or one it needs not. No action starts with K. The rest, up
to J, are in [A to J](lacks-a-j.md). The actions that Zed has are in
[the actions it binds](actions.md) and [the IDE's](actions-ide.md).

| F12 action | F12's keys | Zed action | Status | Note |
| --- | --- | --- | --- | --- |
| `List-selectNextColumnExtendSelection` | Shift+Right |  | n/a | IDE lists |
| `MainMenuButton.ShowMenu` | Alt+. |  | n/a | the menu bar is macOS's |
| `MaintenanceAction` | Shift+Ctrl+Alt+/ |  | n/a | internal |
| `Markdown.Insert` | Alt+Insert |  | n/a | Markdown editor |
| `Markdown.Preview.DecreaseFontSize` | Ctrl+numpad -; Ctrl+- |  | n/a | Markdown editor |
| `Markdown.Styling.CreateLink` | Shift+Ctrl+U |  | n/a | Markdown editor |
| `MiniAiChat.History` | Alt+Down |  | n/a | the AI Assistant's mini chat |
| `ModifyObject` | Ctrl+R, S |  | n/a | database tools |
| `NewRiderProject` | Shift+Ctrl+N |  | n/a | Rider's solution |
| `NewScratchFile` | Shift+Ctrl+Alt+Insert | `workspace::NewFile` | skip | Shift+Ctrl+Alt+Insert: a Mac keyboard has no Insert |
| `NextOccurence` | F8 |  | none | F8 and Shift+F8 go to Next and Previous difference in the editor |
| `OasEndpointsSidePanelSaveAction` | Alt+O |  | n/a | OpenAPI |
| `openAssertEqualsDiff` | Ctrl+D |  | n/a | test diff |
| `OpenInRightSplit` | mouse alt button1 doubleClick |  | none | mouse: Alt+double-click |
| `OpenWinFormsDesignerAction` | Shift+F7 |  | n/a | WinForms |
| `Orchide.GotoInventory` | Shift+Ctrl+O, I |  | n/a | a plugin's |
| `org.intellij.plugins.markdown.ui.actions.styling.InsertImageAction` | Ctrl+U |  | n/a | Markdown editor |
| `org.intellij.plugins.markdown.ui.actions.styling.ToggleStrikethroughAction` | Shift+Ctrl+S |  | n/a | Markdown editor |
| `PasteMultiple` | Shift+Ctrl+V; Shift+Ctrl+Insert |  | none | no clipboard history |
| `PCFindUsagesAction` | Shift+F12 |  | n/a | IDE trees |
| `PCNavigateToSource` | Alt+S |  | n/a | IDE trees |
| `PopupHector` | Shift+Ctrl+Alt+H |  | none |  |
| `PreviousOccurence` | Shift+F8 |  | none | F8 and Shift+F8 go to Next and Previous difference in the editor |
| `PublishGroup.UploadTo` | Shift+Ctrl+Alt+X |  | n/a | deployment |
| `RecentLocations` | Shift+Ctrl+E |  | none | no recent locations; the file finder lists recent files |
| `ReformatWithPrettierAction` | Shift+Ctrl+Alt+P |  | n/a | Prettier |
| `RemoteHostView.EditRemoteFile` | Enter |  | n/a | deployment |
| `RemoteHostView.Rename` | Ctrl+R, R |  | n/a | deployment |
| `ResetColumnsWidth` | Ctrl+Alt+Up |  | n/a | IDE tables |
| `ReSharperNavigateTo` | Alt+backtick; Alt+dead backtick |  | none |  |
| `Rider.Plugins.FSharp.StartFsi` | Ctrl+Alt+F |  | n/a | F# interactive |
| `RiderCodeLens.ShowMore` | Ctrl+K, Ctrl+backtick; Ctrl+K, backtick |  | none |  |
| `RiderDebuggerApplyEncChagnes` | Alt+F10 |  | none |  |
| `RiderDotCoverCoveringTestsPopupCoverAllTestsAction` | Ctrl+K |  | n/a | dotCover |
| `RiderDotCoverCoveringTestsPopupCoverSelectedTestsAction` | Ctrl+C |  | n/a | dotCover |
| `RiderDotCoverCoveringTestsPopupDebugSelectedTestsAction` | Ctrl+D |  | n/a | dotCover |
| `RiderDotCoverCoveringTestsPopupRunAllTestsAction` | Ctrl+R |  | n/a | dotCover |
| `RiderDotCoverCoveringTestsPopupRunSelectedTestsAction` | Ctrl+S |  | n/a | dotCover |
| `RiderDotCoverUnitTestCoverSolutionAction` | Ctrl+U, K; Ctrl+U, Ctrl+K |  | n/a | dotCover |
| `RiderEditSource` | Alt+S |  | n/a | IDE trees |
| `RiderGenerateUnitTestAction` | Ctrl+U, C; Ctrl+U, Ctrl+C |  | n/a | Rider |
| `RiderGoToLinkedTypesAction` | Ctrl+U, G; Ctrl+U, Ctrl+G |  | n/a | Rider |
| `RiderNuGetContextPopupAction` | Alt+Enter; Shift+Enter; Ctrl+Enter |  | n/a | NuGet |
| `RiderNuGetCopyPackageNameAction` | Cmd+C; Ctrl+C |  | n/a | NuGet |
| `RiderNuGetQuickDocAction` | Ctrl+K, I; Ctrl+K, Ctrl+I; Shift+Ctrl+F1 |  | n/a | NuGet |
| `RiderNuGetQuickListPopupAction` | Shift+Alt+N |  | n/a | NuGet |
| `RiderNuGetToggleDependenciesExpanderAction` | Ctrl+D |  | n/a | NuGet |
| `RiderNuGetToggleInfoExpanderAction` | Ctrl+I |  | n/a | NuGet |
| `RiderOpenSolution` | Shift+Ctrl+O |  | n/a | Rider's solution |
| `RiderProblemsViewToolsetContextPopupAction` | Alt+Enter; Shift+Enter; Ctrl+Enter |  | n/a | Rider |
| `RiderReattach` | Shift+Alt+P |  | none |  |
| `RiderUnitTestAppendTestsAction` | Ctrl+U, A; Ctrl+U, Ctrl+A |  | n/a | Rider's unit test window |
| `RiderUnitTestAppendTestsTwAction` | Ctrl+Alt+Insert |  | n/a | Rider's unit test window |
| `RiderUnitTestDebugContextAction` | Ctrl+U, D; Ctrl+U, Ctrl+D |  | none | debug a test from the gutter's runnable |
| `RiderUnitTestDebugContextTwAction` | Ctrl+D |  | n/a | Rider's unit test window |
| `RiderUnitTestDotMemoryUnitContextAction` | Ctrl+U, M; Ctrl+U, Ctrl+M |  | n/a | Rider's unit test window |
| `RiderUnitTestDotMemoryUnitSolutionAction` | Ctrl+U, E; Ctrl+U, Ctrl+E |  | n/a | Rider's unit test window |
| `RiderUnitTestFocusExplorerAction` | Ctrl+Alt+U |  | n/a | Rider's unit test window |
| `RiderUnitTestFocusSessionAction` | Ctrl+Alt+T |  | n/a | Rider's unit test window |
| `RiderUnitTestNewSessionAction` | Ctrl+U, N; Ctrl+U, Ctrl+N |  | n/a | Rider's unit test window |
| `RiderUnitTestNewSessionTwAction` | Shift+Alt+Insert |  | n/a | Rider's unit test window |
| `RiderUnitTestQuickListPopupAction` | Shift+Alt+U |  | n/a | Rider's unit test window |
| `RiderUnitTestRemoveElementsFromSessionTwAction` | Delete |  | n/a | Rider's unit test window |
| `RiderUnitTestRunContextUntilFailAction` | Ctrl+U, W; Ctrl+U, Ctrl+W |  | n/a | Rider's unit test window |
| `RiderUnitTestRunCurrentSessionTwAction` | Ctrl+Y |  | n/a | Rider's unit test window |
| `RiderUnitTestRunSolutionAction` | Ctrl+U, L; Ctrl+U, Ctrl+L |  | n/a | Rider's unit test window |
| `RiderUnitTestRunSolutionTwAction` | Ctrl+L |  | n/a | Rider's unit test window |
| `RiderUnitTestRunTreeTwAction` | Shift+Ctrl+Enter |  | n/a | Rider's unit test window |
| `RiderUnitTestSessionAbortAction` | Ctrl+U, S; Ctrl+U, Ctrl+S |  | n/a | Rider's unit test window |
| `RiderUnitTestSessionRerunFailedTestsAction` | Ctrl+U, F; Ctrl+U, Ctrl+F |  | n/a | Rider's unit test window |
| `RiderUnitTestTreeSelectionPopupAction` | Alt+Enter; Shift+Enter; Ctrl+Enter |  | n/a | Rider's unit test window |
| `RiderUnitTestTreeTextFilterAction` | Cmd+F; Ctrl+F |  | n/a | Rider's unit test window |
| `RunDashboard.CopyConfiguration` | Ctrl+D |  | n/a | services window |
| `RunDashboard.UngroupConfigurations` | Delete |  | n/a | services window |
| `RunInspection` | Shift+Ctrl+Alt+I |  | none |  |
| `RunJsbtTask` | Shift+Ctrl+Alt+F11 |  | n/a | JavaScript |
| `SafeDelete` | Alt+Delete |  | none | Alt+Delete stays Zed's delete to word end |
| `SaveAs` | Shift+Ctrl+S | `workspace::SaveAs` | skip | Shift+Ctrl+S is Save all's too; helix gave it to Save all |
| `SchemaCompare.Cancel` | Alt+Pause |  | n/a | database tools |
| `SchemaCompare.Compare` | Shift+Alt+C |  | n/a | database tools |
| `SchemaCompare.GenerateDeployScript` | Shift+Alt+G |  | n/a | database tools |
| `SchemaCompare.SaveScmpFile` | Ctrl+S |  | n/a | database tools |
| `SchemaCompare.UpdateTarget` | Shift+Alt+U |  | n/a | database tools |
| `ScrollTreeToCenter` | Ctrl+M, C; Ctrl+M, Ctrl+C |  | n/a | IDE trees |
| `SearchEverywhere.CompleteCommand` | Tab |  | n/a | IDE popup: Zed's pickers have their own keys |
| `SearchEverywhere.NavigateToNextGroup` | PgDn; Ctrl+Down |  | n/a | IDE popup: Zed's pickers have their own keys |
| `SearchEverywhere.NavigateToPrevGroup` | PgUp; Ctrl+Up |  | n/a | IDE popup: Zed's pickers have their own keys |
| `SearchEverywhere.NextTab` | Tab |  | n/a | IDE popup: Zed's pickers have their own keys |
| `SearchEverywhere.PrevTab` | Shift+Tab |  | n/a | IDE popup: Zed's pickers have their own keys |
| `SearchEverywhere.SelectItem` | Enter |  | n/a | IDE popup: Zed's pickers have their own keys |
| `SelectVirtualTemplateElement` | Shift+Alt+O |  | none |  |
| `ServiceView.GroupByContributor` | Ctrl+Alt+T |  | n/a | services window |
| `ServiceView.JumpToServices` | Ctrl+F2 |  | n/a | services window |
| `Session.Rename` | Ctrl+R, R | `terminal::RenameTerminal` | skip | Ctrl+R must reach the shell, so no Ctrl+R chord |
| `ShelveChanges.UnshelveWithDialog` | Shift+Ctrl+U |  | n/a | shelves |
| `ShowContent` | Alt+Down |  | n/a | IDE UI |
| `ShowExecutionPoint` | `Alt+numpad *` |  | none |  |
| `ShowFilterPopup` | Ctrl+Alt+F |  | n/a | IDE trees |
| `ShowNavBar` | Ctrl+F2 |  | none | Ctrl+F2 is macOS's too |
| `ShowSearchHistory` | Alt+Down |  | n/a | IDE UI |
| `ShowUmlDiagram` | Shift+Ctrl+Alt+U |  | n/a | diagrams |
| `ShowUmlDiagramPopup` | Ctrl+Alt+U |  | n/a | diagrams |
| `SplitChooser.NextWindow` | Tab |  | n/a | IDE UI |
| `SplitChooser.PreviousWindow` | Shift+Tab |  | n/a | IDE UI |
| `SplitChooser.Split` | Enter |  | n/a | IDE UI |
| `SplitChooser.SplitCenter` | Space |  | n/a | IDE UI |
| `sql.ExtractFunctionAction` | Ctrl+R, M |  | n/a | database tools |
| `sql.SelectInDatabaseView` | Shift+Alt+B |  | n/a | database tools |
| `StopBackgroundProcesses` | Shift+Ctrl+F2 |  | none |  |
| `StretchSplitToBottom` | Ctrl+Alt+Down |  | none | Zed resizes splits by mouse, or in vim mode |
| `StretchSplitToLeft` | Ctrl+Alt+Left |  | none | Zed resizes splits by mouse, or in vim mode |
| `StretchSplitToRight` | Ctrl+Alt+Right |  | none | Zed resizes splits by mouse, or in vim mode |
| `StretchSplitToTop` | Ctrl+Alt+Up |  | none | Zed resizes splits by mouse, or in vim mode |
| `SurroundWithLiveTemplate` | Ctrl+K, S; Ctrl+K, Ctrl+S |  | none | Zed has no surround outside vim mode |
| `SwitchCodeToDesigner` | Shift+F7 |  | n/a | WinForms |
| `SwitchCoverage` | Ctrl+Alt+F6 |  | n/a | coverage |
| `SwitchHeaderSource` | Alt+O; Ctrl+K, O; Ctrl+K, Ctrl+O |  | none | editor::SwitchSourceHeader serves clangd alone |
| `Table-selectNextColumnExtendSelection` | Shift+Right |  | n/a | IDE tables |
| `Table-selectPreviousColumnExtendSelection` | Shift+Left |  | n/a | IDE tables |
| `TableResult.GrowSelection` | Shift+Alt+=; Ctrl+W |  | n/a | database tools |
| `TableResult.SelectAllOccurrences` | Shift+Alt+; |  | n/a | database tools |
| `TableResult.SelectColumn` | Shift+Alt+=; Ctrl+W |  | n/a | database tools |
| `TableResult.SelectNextOccurrence` | Shift+Alt+. |  | n/a | database tools |
| `TableResult.ShrinkSelection` | Shift+Alt+-; Shift+Ctrl+W |  | n/a | database tools |
| `TableResult.UnselectPreviousOccurrence` | Shift+Alt+, |  | n/a | database tools |
| `tasks.open.in.browser` | Shift+Alt+B |  | n/a | IDE tasks and contexts |
| `tasks.switch` | Shift+Alt+T |  | n/a | IDE tasks and contexts |
| `Terminal.ExpandBlockSelectionAbove` | Shift+Up |  | n/a | the JetBrains terminal's blocks |
| `Terminal.ExpandBlockSelectionBelow` | Shift+Down |  | n/a | the JetBrains terminal's blocks |
| `Terminal.GenerateCommandFromText` | Ctrl+/ | `assistant::InlineAssist` | skip | Ctrl+/ reaches the shell; Zed's Ctrl+Enter opens the terminal's inline assistant |
| `Terminal.SelectBlockAbove` | Up; Ctrl+Up |  | n/a | the JetBrains terminal's blocks |
| `Terminal.SelectBlockBelow` | Down |  | n/a | the JetBrains terminal's blocks |
| `Terminal.SelectLastBlock` | Ctrl+Up |  | n/a | the JetBrains terminal's blocks |
| `Terminal.SelectPrompt` | Ctrl+Down |  | n/a | the JetBrains terminal's blocks |
| `Terminal.ShowDocumentation` | Ctrl+K, I; Ctrl+K, Ctrl+I; Shift+Ctrl+F1 |  | n/a | the JetBrains terminal's blocks |
| `TodoViewGroupByFlattenPackage` | Ctrl+Alt+C |  | n/a | TODO window |
| `TodoViewGroupByShowModules` | Ctrl+Alt+M |  | n/a | TODO window |
| `TodoViewGroupByShowPackages` | Ctrl+Alt+P |  | n/a | TODO window |
| `ToggleRenderedDocPresentation` | Ctrl+K, V; Ctrl+K, Ctrl+V |  | none |  |
| `ToggleTemporaryLineBreakpoint` | Shift+Ctrl+Alt+F8 |  | none |  |
| `Tree-selectParentExtendSelection` | Shift+Left |  | n/a | IDE trees |
| `UML.Find` | Cmd+F; Ctrl+F |  | n/a | diagrams |
| `Uml.NewElement` | Ctrl+N; Alt+Insert |  | n/a | diagrams |
| `Uml.NodeIntentions` | Alt+Enter; Shift+Enter; Ctrl+Enter |  | n/a | diagrams |
| `UML.ShowChanges` | Shift+Ctrl+Alt+D |  | n/a | diagrams |
| `Uml.ShowDiff` | Shift+Ctrl+D |  | n/a | diagrams |
| `UML.ShowStructure` | Alt+\ |  | n/a | diagrams |
| `UpdateRunningApplication` | Ctrl+F10 |  | none |  |
| `UsageFiltering.WriteAccess` | Ctrl+W |  | n/a | usages window |
| `UsageGrouping.Directory` | Ctrl+Alt+P |  | n/a | usages window |
| `UsageGrouping.FileStructure` | Ctrl+Alt+F |  | n/a | usages window |
| `UsageGrouping.FlattenModules` | Ctrl+Alt+O |  | n/a | usages window |
| `UsageGrouping.Module` | Ctrl+Alt+M |  | n/a | usages window |
| `UsageGrouping.UsageType` | Ctrl+Alt+T |  | n/a | usages window |
| `UsageView.Exclude` | Delete |  | n/a | usages window |
| `UsageView.Include` | Insert |  | n/a | usages window |
| `Vcs.CombinedDiff.CaretToNextBlock` | Down; Right; PgDn |  | n/a | diff viewer |
| `Vcs.CombinedDiff.CaretToPrevBlock` | Up; Left; PgUp |  | n/a | diff viewer |
| `Vcs.CombinedDiff.ToggleCollapseBlock` | Ctrl+Esc |  | n/a | diff viewer |
| `Vcs.Commit.CloseDialog` | Esc |  | n/a | commit dialog |
| `Vcs.EditSource` | Alt+S |  | n/a | IDE trees |
| `Vcs.Log.GoToChild` | Left |  | n/a | the VCS log's tree |
| `Vcs.Log.GoToParent` | Right |  | n/a | the VCS log's tree |
| `Vcs.Log.GoToRef` | Cmd+F; Ctrl+F |  | n/a | the VCS log's tree |
| `Vcs.Log.ShowAllAffected` | Shift+Alt+A |  | n/a | the VCS log's tree |
| `Vcs.Log.ShowTooltip` | Ctrl+K, I; Ctrl+K, Ctrl+I; Shift+Ctrl+F1 |  | n/a | the VCS log's tree |
| `Vcs.MoveChangedLinesToChangelist` | Shift+Alt+M |  | n/a | changelists |
| `Vcs.ReformatCommitMessage` | Ctrl+Alt+Enter; Ctrl+K, F; Ctrl+K, Ctrl+F; Ctrl+K, D; Ctrl+K, Ctrl+D |  | n/a | commit dialog |
| `Vcs.Shelf.Drop` | Delete |  | n/a | shelves |
| `Vcs.ShowMessageHistory` | Ctrl+H |  | n/a | commit dialog |
| `VcsHistory.ShowAllAffected` | Shift+Alt+A |  | n/a | VCS history |
| `Voice.ActivateVoiceAction` | Shift+Alt+E |  | n/a | voice input |
| `Voice.ReferenceEntityInChatAction` | Shift+Alt+A |  | n/a | voice input |
| `WD.UploadCurrentRemoteFileAction` | Shift+Alt+Q |  | n/a | deployment |
| `WebOpenInAction` | Alt+F2; Ctrl+Alt+F2 |  | n/a | browsers |
| `WelcomeScreen.RemoveSelected` | Delete |  | n/a | IDE welcome screen |
| `XDebugger.CopyWatch` | Ctrl+D |  | n/a | debugger trees |
| `XDebugger.JumpToTypeSource` | Shift+F4 |  | n/a | debugger trees |
| `XDebugger.MoveWatchDown` | Alt+Down |  | n/a | debugger trees |
| `XDebugger.MoveWatchUp` | Alt+Up |  | n/a | debugger trees |
| `XDebugger.NewWatch` | Insert | `variable_list::AddWatch` | skip | Insert: a Mac keyboard has none |
| `XPathView.Actions.Evaluate` | Ctrl+Alt+X, E |  | n/a | XPath |
| `XPathView.Actions.FindByExpression` | Ctrl+Alt+X, F |  | n/a | XPath |
| `XPathView.Actions.ShowPath` | Ctrl+Alt+X, P |  | n/a | XPath |
