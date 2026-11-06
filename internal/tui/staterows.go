package tui

import (
	"strings"
	"unicode"

	"github.com/zadykian/ws/step"
)

// The tabs of the state screen, in their order.
const (
	packagesTab = iota
	servicesTab
	filesTab
	conflictsTab
	pendingTab
	tabCount
)

// tabNames holds each tab's title.
var tabNames = [tabCount]string{"Packages", "Services", "Files", "Conflicts", "Pending"}

// itemTabs says which tab lists an item of each kind. A check has none: it stands against ws by
// what it is, so the Conflicts tab lists it.
var itemTabs = map[step.ItemKind]int{
	step.PackageItem: packagesTab,
	step.ServiceItem: servicesTab,
	step.FileItem:    filesTab,
	step.LinkItem:    filesTab,
	step.DirItem:     filesTab,
}

// StateSection is a part of what a machine reports: a role's steps, or the Mac's, which have no name.
type StateSection struct {
	Name    string
	Reports []step.Report
}

// stateFact is a line of the detail pane.
type stateFact struct{ label, text string }

// stateRow is a line of a tab and what the detail pane shows of it, all of it safe to draw.
type stateRow struct {
	name, step string
	facts      []stateFact
	conflict   string
	diff       string
}

// tabRows sorts what sections report into the tabs. Each tab keeps the order of a run, so that a
// role's rows stand together. An item that stands against ws shows under its kind and in Conflicts.
// A step left unread goes under Pending, as a change before it holds it back.
func tabRows(sections []StateSection) [tabCount][]stateRow {
	var tabs [tabCount][]stateRow
	for _, section := range sections {
		for _, report := range section.Reports {
			name := stepPath(section.Name, report.Step)
			for _, item := range report.Items {
				row := itemRow(name, item)
				if tab, ok := itemTabs[item.Kind]; ok {
					tabs[tab] = append(tabs[tab], row)
				}
				if item.Conflict != "" {
					tabs[conflictsTab] = append(tabs[conflictsTab], row)
				}
			}
			for _, change := range report.Changes {
				tabs[pendingTab] = append(tabs[pendingTab], pendingRow(name, change))
			}
			if report.After != "" {
				tabs[pendingTab] = append(tabs[pendingTab], unreadRow(name, report.After))
			}
		}
	}
	return tabs
}

// stepPath names a step as --only does.
func stepPath(section, name string) string {
	if section == "" {
		return name
	}
	return section + "/" + name
}

func itemRow(path string, item step.Item) stateRow {
	return stateRow{
		name: firstLine(printable(item.Name)),
		step: path,
		facts: []stateFact{
			{"Kind", string(item.Kind)},
			{"Step", printable(path)},
			{"Want", printable(item.Want)},
			{"Found", printable(item.Found)},
		},
		conflict: printable(item.Conflict),
		diff:     printable(item.Diff),
	}
}

// pendingRow takes a change's first line as its name, as the plain report does. What follows it is
// the file's diff, where the change has one.
func pendingRow(path string, change step.Pending) stateRow {
	name, diff, _ := strings.Cut(printable(change.Text), "\n")
	facts := []stateFact{{"Kind", string(change.Kind)}, {"Step", printable(path)}}
	return stateRow{name: name, step: path, facts: facts, diff: diff}
}

func unreadRow(path, after string) stateRow {
	facts := []stateFact{{"Not read", "it waits on " + printable(after)}}
	return stateRow{name: firstLine(printable(path)), step: path, facts: facts}
}

// firstLine returns the text up to its first line's end.
func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return line
}

// printable makes text safe to draw. A pane measures columns, which a tab has none of, and what a
// file holds must not reach the terminal as a control code.
func printable(text string) string {
	text = strings.ReplaceAll(strings.ToValidUTF8(text, "�"), "\t", "    ")
	return strings.Map(func(r rune) rune {
		if r != '\n' && unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
}
