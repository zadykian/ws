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
