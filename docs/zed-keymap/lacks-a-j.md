# The actions the Zed keymap lacks, A to J

Each row is an F12 action that Zed's keymap binds no key for. The status `none` means
Zed lacks the action. The status `skip` means Zed has it, but the binding is left out. The
status `n/a` means a tool Zed lacks, or one it needs not. The rest, from L, are in
[L to Z](lacks-l-z.md). The actions that Zed has are in [the actions it binds](actions.md) and
[the IDE's](actions-ide.md).

| F12 action | F12's keys | Zed action | Status | Note |
| --- | --- | --- | --- | --- |
| `ActivateBuildToolWindow` | F10 |  | none | no test or build window: tasks run in the terminal |
| `ActivateNuGetToolWindow` | F4; Alt+4 |  | n/a | no NuGet window |
| `ActivateTestsToolWindow` | F3 |  | none | no test or build window: tasks run in the terminal |
| `ActivateUnitTestsToolWindow` | F3; Alt+3 |  | none | no test or build window: tasks run in the terminal |
| `AddRiderItem` | Shift+Alt+A |  | n/a | Rider's solution |
| `AIAssistant.Chat.AIPopupChat` | Shift+Ctrl+Alt+\ |  | n/a | the AI Assistant's chat and popups: Zed's agent panel has keys of its own |
| `AIAssistant.Chat.SendActions.NewLine` | Shift+Enter |  | n/a | the AI Assistant's chat and popups: Zed's agent panel has keys of its own |
| `AIAssistant.Chat.SendActions.SendToNewChat` | Shift+Ctrl+Enter |  | n/a | the AI Assistant's chat and popups: Zed's agent panel has keys of its own |
| `AIAssistant.Chat.Toggle.Chat.Mode` | Ctrl+Alt+/; Ctrl+Alt+numpad /; Ctrl+K, C; Ctrl+K, Ctrl+C; Ctrl+K, Ctrl+/; Ctrl+K, Ctrl+U |  | n/a | the AI Assistant's chat and popups: Zed's agent panel has keys of its own |
| `AIAssistant.CodeGeneration.Actions.ShowIntentionActions` | Alt+Enter; Shift+Enter; Ctrl+Enter |  | n/a | the AI Assistant's chat and popups: Zed's agent panel has keys of its own |
| `AIAssistant.CodeGeneration.Actions.Specify` | Ctrl+/ |  | n/a | the AI Assistant's chat and popups: Zed's agent panel has keys of its own |
| `AIAssistant.ToolWindow.Chat.Cancel.Selection` | Esc |  | n/a | the AI Assistant's chat and popups: Zed's agent panel has keys of its own |
| `AIAssistant.ToolWindow.Chat.Copy.Chameleon` | Cmd+C; Ctrl+C |  | n/a | the AI Assistant's chat and popups: Zed's agent panel has keys of its own |
| `AIAssistant.ToolWindow.Chat.Custom.Select.All` | Cmd+A; Ctrl+A |  | n/a | the AI Assistant's chat and popups: Zed's agent panel has keys of its own |
| `AIAssistant.ToolWindow.Chat.Delete` | Delete |  | n/a | the AI Assistant's chat and popups: Zed's agent panel has keys of its own |
| `AIAssistant.ToolWindow.Chat.Find` | Cmd+F; Ctrl+F |  | n/a | the AI Assistant's chat and popups: Zed's agent panel has keys of its own |
| `AIAssistant.ToolWindow.NewChatAction` | Ctrl+N; Alt+Insert |  | n/a | the AI Assistant's chat and popups: Zed's agent panel has keys of its own |
| `AIAssistant.ToolWindow.NewChatActionAlt` | Ctrl+N; Alt+Insert |  | n/a | the AI Assistant's chat and popups: Zed's agent panel has keys of its own |
| `AIAssistant.ToolWindow.RenameDialog` | Ctrl+R, R |  | n/a | the AI Assistant's chat and popups: Zed's agent panel has keys of its own |
| `Arrangement.Alias.Rule.Remove` | Delete |  | n/a | IDE settings UI |
| `Arrangement.Rule.Group.Condition.Move.Up` | Alt+Up |  | n/a | IDE settings UI |
| `Arrangement.Rule.Match.Condition.Move.Down` | Alt+Down |  | n/a | IDE settings UI |
| `Arrangement.Rule.Match.Condition.Move.Up` | Alt+Up |  | n/a | IDE settings UI |
| `Arrangement.Rule.Remove` | Delete |  | n/a | IDE settings UI |
| `AssemblyDiffAction` | Ctrl+D |  | n/a | Rider |
| `BookmarksView.Delete` | Delete |  | n/a | IDE trees |
| `BraceOrQuoteOut` | Tab |  | none | no overwrite mode, column mode or complete statement |
| `CancelBuildAction` | Ctrl+F9; Ctrl+Break |  | none |  |
| `ChangesView.GroupBy.Module` | Ctrl+Alt+M |  | n/a | changelists |
| `ChangesView.Move` | Shift+Alt+M |  | n/a | changelists |
| `ChangesView.SetDefault` | Ctrl+Space |  | n/a | changelists |
| `ChangeTypeSignature` | Shift+Ctrl+F6 |  | none |  |
| `CIDR.Debugger.DisassembleFrame` | Ctrl+D |  | n/a | C++ debugger |
| `CIDR.Debugger.ImageViewerActions.CopyImage` | Cmd+C; Ctrl+C |  | n/a | C++ debugger |
| `CloseGotItTooltip` | Esc |  | n/a | IDE UI |
| `Code.Review.Editor.New.Comment` | Shift+Ctrl+X |  | n/a | code review |
| `CodeFloatingToolbar.GotoNextMenu` | Tab |  | n/a | IDE UI |
| `CodeFloatingToolbar.GotoPrevMenu` | Shift+Tab |  | n/a | IDE UI |
| `CodeReview.NextComment` | F8 |  | n/a | code review |
| `CodeReview.PreviousComment` | Shift+F8 |  | n/a | code review |
| `CollapseExpandableComponent` | Ctrl+numpad -; Ctrl+- |  | n/a | IDE UI |
| `CollapseTreeNode` | numpad - |  | skip | numpad keys: the arrows serve |
| `Compare.SameVersion` | Ctrl+D |  | n/a | diff viewer |
| `CompareTwoFiles` | Ctrl+D |  | n/a | diff viewer |
| `Compile` | Ctrl+F7 |  | n/a | Java and C++ builds |
| `CompileFile` | Ctrl+F7 |  | n/a | Java and C++ builds |
| `Console.DbmsOutput` | Ctrl+B |  | n/a | database tools |
| `Console.Execute` | Enter |  | n/a | database tools |
| `Console.Execute.Multiline` | Ctrl+Enter |  | n/a | database tools |
| `Console.Jdbc.Cancel` | Alt+F5 |  | n/a | database tools |
| `Console.Jdbc.Execute` | Ctrl+Enter |  | n/a | database tools |
| `Console.TableResult.AddColumn` | Shift+Alt+Insert |  | n/a | database tools |
| `Console.TableResult.AddRow` | Alt+Insert |  | n/a | database tools |
| `Console.TableResult.CloneColumn` | Shift+Ctrl+Alt+D |  | n/a | database tools |
| `Console.TableResult.CloneRow` | Ctrl+D |  | n/a | database tools |
| `Console.TableResult.ColumnsList` | Alt+\ |  | n/a | database tools |
| `Console.TableResult.ColumnSortAsc` | Alt+Up |  | n/a | database tools |
| `Console.TableResult.ColumnSortDesc` | Alt+Down |  | n/a | database tools |
| `Console.TableResult.ColumnSortReset` | Shift+Ctrl+Alt+Backspace |  | n/a | database tools |
| `Console.TableResult.CompareCells` | Shift+Ctrl+D |  | n/a | database tools |
| `Console.TableResult.Copy` | Cmd+C; Ctrl+C |  | n/a | database tools |
| `Console.TableResult.DeleteColumns` | Shift+Alt+Delete |  | n/a | database tools |
| `Console.TableResult.EditFilterCriteria` | Shift+Ctrl+Alt+F |  | n/a | database tools |
| `Console.TableResult.EditValue` | Enter; Alt+Enter |  | n/a | database tools |
| `Console.TableResult.FindInGrid` | Cmd+F; Ctrl+F |  | n/a | database tools |
| `Console.TableResult.GotoReferencedResult` | Enter; Alt+Enter; mouse button1 doubleClick |  | n/a | database tools |
| `Console.TableResult.GotoReferencingResult` | Alt+D |  | n/a | database tools |
| `Console.TableResult.MaximizeEditingCell` | Shift+Ctrl+Alt+M |  | n/a | database tools |
| `Console.TableResult.NavigateAction` | Alt+S |  | n/a | database tools |
| `Console.TableResult.OpenLocalFileAction` | Alt+S |  | n/a | database tools |
| `Console.TableResult.OpenWebUrlAction` | Alt+S |  | n/a | database tools |
| `Console.TableResult.RevertSelected` | Ctrl+Alt+Z |  | n/a | database tools |
| `Console.TableResult.SelectColumn` | Ctrl+Space |  | n/a | database tools |
| `Console.TableResult.SelectRow` | Shift+Space |  | n/a | database tools |
| `Console.TableResult.SetNull` | Ctrl+Alt+N |  | n/a | database tools |
| `Console.TableResult.ShowRecordView` | Shift+Ctrl+Enter |  | n/a | database tools |
| `Console.TableResult.Submit` | Ctrl+Enter |  | n/a | database tools |
| `Console.TableResult.SubmitAndCommit` | Shift+Ctrl+Alt+Enter |  | n/a | database tools |
| `Console.Transaction.Commit` | Shift+Ctrl+Alt+Enter |  | n/a | database tools |
| `context.clear` | Shift+Alt+X |  | n/a | IDE tasks and contexts |
| `context.load` | Shift+Alt+L |  | n/a | IDE tasks and contexts |
| `context.save` | Shift+Alt+S |  | n/a | IDE tasks and contexts |
| `CppIncludesHierarchy` | Shift+Alt+H |  | n/a | C++ |
| `DatabaseView.CopyAction` | Ctrl+D |  | n/a | database tools |
| `DatabaseView.CopyDdlAction` | Shift+Ctrl+Alt+G |  | n/a | database tools |
| `DatabaseView.DataSourceCopyAction` | Cmd+C; Ctrl+C |  | n/a | database tools |
| `DatabaseView.DataSourceCutAction` | Ctrl+X; Cmd+X |  | n/a | database tools |
| `DatabaseView.DataSourcePasteAction` | Cmd+V; Ctrl+V |  | n/a | database tools |
| `DatabaseView.Ddl.AlterObject` | Ctrl+R, S |  | n/a | database tools |
| `DatabaseView.DropAction` | Delete |  | n/a | database tools |
| `DatabaseView.FullTextSearch` | Shift+Ctrl+Alt+F |  | n/a | database tools |
| `DatabaseView.OpenDdlInConsole` | Shift+Ctrl+Alt+B |  | n/a | database tools |
| `DatabaseView.ShowDiff` | Ctrl+D |  | n/a | database tools |
| `DatabaseView.SqlGenerator` | Ctrl+Alt+G |  | n/a | database tools |
| `DatabaseView.Tools` | Alt+Enter; Shift+Enter; Ctrl+Enter |  | n/a | database tools |
| `DatabaseView.Tools.RevertChanges` | Ctrl+Alt+Z |  | n/a | database tools |
| `DatabaseView.Tools.SubmitChanges` | Ctrl+Alt+K |  | n/a | database tools |
| `Debugger.MarkObject` | Ctrl+K, K; Ctrl+K, Ctrl+K |  | none |  |
| `DecreaseColumnWidth` | Ctrl+Alt+Left |  | n/a | IDE tables |
| `Diagram.DeleteSelection` | Delete |  | n/a | diagrams |
| `Diagram.DeselectAll` | Ctrl+Alt+A |  | n/a | diagrams |
| `Diagram.SelectAll` | Cmd+A; Ctrl+A |  | n/a | diagrams |
| `Diff.ApplyRightSide` | Ctrl+Alt+A |  | n/a | diff viewer |
| `Diff.FocusOppositePane` | Ctrl+\, Ctrl+Tab; Ctrl+\, Tab; Shift+Ctrl+Tab |  | n/a | diff viewer |
| `Diff.ShowDiff` | Ctrl+D |  | n/a | diff viewer |
| `Diff.ShowSettingsPopup` | Shift+Ctrl+D |  | n/a | diff viewer |
| `DirDiffMenu.SynchronizeDiff` | Enter |  | n/a | diff viewer |
| `Docker.RemoteServers.StopComposeApp` | Alt+F5 |  | n/a | Docker window |
| `Docker.RemoteServers.StopDeploy` | Alt+F5 |  | n/a | Docker window |
| `Documentation.EditSource` | Alt+S |  | n/a | IDE UI |
| `DomCollectionControl.Add` | Insert |  | n/a | IDE settings UI |
| `DomCollectionControl.Edit` | Alt+S |  | n/a | IDE settings UI |
| `DomCollectionControl.Remove` | Delete |  | n/a | IDE settings UI |
| `DumpLookupElementWeights` | Shift+Ctrl+Alt+W |  | n/a | internal |
| `DumpMLCompletionFeatures` | Shift+Ctrl+Alt+9 |  | n/a | internal |
| `EditorChooseLookupItemDot` | Ctrl+. |  | none |  |
| `EditorCompleteStatement` | Shift+Ctrl+Enter |  | none | no overwrite mode, column mode or complete statement |
| `EditorFocusGutter` | Shift+Alt+6, F |  | none |  |
| `EditorPasteFromX11` | mouse button2 |  | n/a | X11's middle-click paste |
| `EditorShowGutterIconTooltip` | Shift+Alt+6, T |  | none |  |
| `EditorToggleColumnMode` | Shift+Alt+Insert |  | none | no overwrite mode, column mode or complete statement |
| `EditorToggleInsertState` | Insert |  | none | no overwrite mode, column mode or complete statement |
| `EditPropertiesAction` | Alt+Enter; Shift+Enter; Ctrl+Enter |  | n/a | IDE trees |
| `EditSource` | Alt+S |  | n/a | IDE trees |
| `EmmetNextEditPoint` | Shift+Alt+] |  | n/a | Emmet |
| `EmmetPreviousEditPoint` | Shift+Alt+[ |  | n/a | Emmet |
| `EnableDaemon` | Shift+Ctrl+Alt+8 |  | n/a | internal |
| `ExpandExpandableComponent` | Ctrl+numpad + |  | n/a | IDE UI |
| `ExpandTreeNode` | numpad + |  | skip | numpad keys: the arrows serve |
| `ExportToTextFile` | Alt+O |  | n/a |  |
| `ExternalSystem.DetachProject` | Delete |  | n/a | IDE trees |
| `FileChooser.Delete` | Delete |  | n/a | IDE file dialog |
| `FileChooser.GoBackward` | Ctrl+-; Ctrl+numpad - |  | n/a | IDE file dialog |
| `FileChooser.GotoDesktop` | Ctrl+D |  | n/a | IDE file dialog |
| `FileChooser.GoToParent` | Backspace |  | n/a | IDE file dialog |
| `FileChooser.GoToRoot` | Ctrl+\ |  | n/a | IDE file dialog |
| `FileChooser.NewFolder` | Alt+Insert; Ctrl+N |  | n/a | IDE file dialog |
| `FileChooser.TogglePathBar` | Ctrl+P |  | n/a | IDE file dialog |
| `FullyExpandTreeNode` | `numpad *` |  | skip | numpad keys: the arrows serve |
| `Generate.Missing.Members.ES6` | Shift+Ctrl+I |  | n/a | JavaScript |
| `Generate.Missing.Members.TypeScript` | Shift+Ctrl+I |  | n/a | JavaScript |
| `Git.Log.Branches.Navigate.Log.To.Selected.Branch` | Alt+F1 |  | n/a | the VCS log's tree |
| `Git.Stage.Reset` | Ctrl+Alt+Z | `git::UnstageFile` | skip | Ctrl+Alt+Z is Revert's in the git panel |
| `Git.Stash.Drop` | Delete |  | n/a | IDE trees |
| `Git.WorkingTrees.Refresh` | Ctrl+Alt+Y |  | n/a | worktrees window |
| `Git.WorkingTrees.Remove` | Delete |  | n/a | worktrees window |
| `GoOpenSettings` | Alt+S |  | n/a | IDE trees |
| `GoShareInPlaygroundAction` | Shift+Ctrl+Alt+S |  | n/a | Go playground |
| `GoToControllerOrView` | Alt+O; Ctrl+K, O; Ctrl+K, Ctrl+O |  | none | editor::SwitchSourceHeader serves clangd alone |
| `GotoCustomRegion` | Ctrl+Alt+. |  | none |  |
| `GotoRelated` | Ctrl+Alt+F7 |  | none | editor::SwitchSourceHeader serves clangd alone |
| `Graph.Delete` | Delete |  | n/a | diagrams |
| `Graph.ZoomIn` | numpad + |  | n/a | diagrams |
| `Graph.ZoomOut` | numpad - |  | n/a | diagrams |
| `Hg.Commit.And.Push.Executor` | Shift+Ctrl+Alt+K |  | n/a | Mercurial |
| `hg4idea.QGotoFromPatches` | Shift+Alt+G |  | n/a | Mercurial |
| `hg4idea.QPushAction` | Shift+Alt+P |  | n/a | Mercurial |
| `HTTPClient.PreviewHtml.NavigateBack` | Left |  | n/a | HTTP client |
| `HTTPClient.PreviewHtml.NavigateForward` | Right |  | n/a | HTTP client |
| `Images.EditExternally` | Ctrl+Alt+F4 |  | n/a | image editor |
| `Images.Editor.ToggleGrid` | Ctrl+' |  | n/a | image editor |
| `Images.Editor.ZoomIn` | Ctrl+numpad + |  | n/a | image editor |
| `Images.Editor.ZoomOut` | Ctrl+numpad -; Ctrl+- |  | n/a | image editor |
| `Images.Thumbnails.Hide` | Ctrl+F4 |  | n/a | image editor |
| `Images.Thumbnails.ToggleRecursive` | `Alt+numpad *` |  | n/a | image editor |
| `Images.Thumbnails.UpFolder` | Backspace |  | n/a | image editor |
| `IncreaseColumnWidth` | Ctrl+Alt+Right |  | n/a | IDE tables |
| `InlinePromptGenerateCodeAction` | Tab | `assistant::InlineAssist` | skip | Tab accepts an edit prediction; Ctrl+/ opens the inline assistant |
| `InspectThis` | Shift+Ctrl+Alt+A |  | none |  |
| `JavaScript.ShowComponentUsages` | Shift+Ctrl+D |  | n/a | JavaScript |
| `Jdbc.OpenConsole.New` | Shift+Ctrl+Q |  | n/a | database tools |
| `Jdbc.OpenConsole.New.Generate` | Shift+Ctrl+Q |  | n/a | database tools |
| `Jdbc.OpenConsole.New.InPath` | Shift+Ctrl+Alt+Q |  | n/a | database tools |
| `Jdbc.OpenConsole.Scratch` | Shift+Ctrl+Alt+Q |  | n/a | database tools |
| `Jdbc.OpenEditor.Console` | Alt+S |  | n/a | database tools |
| `Jdbc.OpenEditor.DDL` | Alt+D |  | n/a | database tools |
| `Jdbc.OpenEditor.Grid.DDL` | Ctrl+Alt+F7 |  | n/a | database tools |
| `JsonPathExportEvaluateResultAction` | Alt+O |  | n/a | JSONPath |
| `JumpToLastWindow` | Ctrl+Alt+Backspace; Shift+Alt+F6 |  | none |  |
| `JumpToStatement` | Shift+Ctrl+F10 |  | none |  |
