package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// An origin is where a mirror's line comes from: the source's line, and the rune column at which
// the mirrored text starts in it. The zero origin is a blank line's, which comes from no comment.
type origin struct{ line, col int }

// A mirror is a file's comments as Markdown, a line for each comment line, with its origin. A
// blank line ends a comment, so that Vale reads two comments as two paragraphs. file is the
// source's path, and texts counts the lines that hold text.
type mirror struct {
	file  string
	lines []string
	from  []origin
	texts int
}

// newMirror starts with a blank line, lest a first comment of dashes start front matter.
func newMirror() *mirror {
	return &mirror{lines: []string{""}, from: []origin{{}}}
}

// fence matches a line that opens a fenced code block, whose text Vale skips to the fence's end.
var fence = regexp.MustCompile("^ {0,3}(?:```|~~~)")

// add takes a comment's line. A fence or an HTML tag in it would hide the comments after it from
// Vale. So the fence's marks and each < become runes that Vale reads as text, one for one.
func (m *mirror) add(text string, o origin) {
	if loc := fence.FindStringIndex(text); loc != nil {
		text = text[:loc[1]-3] + "xxx" + text[loc[1]:]
	}
	text = strings.ReplaceAll(text, "<", "‹")
	m.lines = append(m.lines, text)
	m.from = append(m.from, o)
	if strings.TrimSpace(text) != "" {
		m.texts++
	}
}

// gap ends the current paragraph.
func (m *mirror) gap() {
	if m.lines[len(m.lines)-1] != "" {
		m.add("", origin{})
	}
}

// source maps a line and a rune column of the mirror, each from 1, to those of the source.
func (m *mirror) source(line, col int) (origin, bool) {
	if m == nil || line < 1 || line > len(m.from) || m.from[line-1].line == 0 {
		return origin{}, false
	}
	return origin{m.from[line-1].line, m.from[line-1].col + col - 1}, true
}

var (
	// goDirective matches a Go directive's body, such as //nolint's, as go/ast tells one.
	goDirective = regexp.MustCompile(`^(?:[a-z0-9]+:[a-z0-9]|line |extern |export |\s*\+build)`)
	// hashDirective matches the body of ShellCheck's, yamllint's and zizmor's directives, and not
	// a comment's sentence that starts with the linter's name.
	hashDirective = regexp.MustCompile(`^\s*(?:shellcheck\s+[a-z-]+=|yamllint\s+(?:disable|enable)|zizmor:)`)
)

// text returns the text that a comment's body, after its marker, gives Vale, and the runes of the
// body before it. A directive gives the reason after its own marker alone, and false without one.
func text(body string, directive *regexp.Regexp, marker string) (string, int, bool) {
	skip := 0
	if directive != nil && directive.MatchString(body) {
		i := strings.Index(body, " "+marker)
		if i < 0 {
			return "", 0, false
		}
		skip = utf8.RuneCountInString(body[:i]) + 1 + len(marker)
		body = body[i+1+len(marker):]
	}
	rest := strings.TrimPrefix(body, " ")
	return rest, skip + len(body) - len(rest), true
}

// goMirror mirrors a Go file's comments, as go/parser finds them, trailing ones included.
func goMirror(src []byte) (*mirror, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(src), "\n")
	names := docNames(f)
	m := newMirror()
	for _, group := range f.Comments {
		m.gap()
		for i, c := range group.List {
			p := fset.Position(c.Slash)
			col := utf8.RuneCountInString(lines[p.Line-1][:p.Column-1]) + 1
			// The mirror skips a block comment, as ws writes none.
			body, skip, ok := text(c.Text[2:], goDirective, "//")
			if strings.HasPrefix(c.Text, "/*") || !ok {
				m.gap()
				continue
			}
			if i == 0 {
				body = masked(body, names[group])
			}
			m.add(body, origin{p.Line, col + 2 + skip})
		}
	}
	return m, nil
}

// docNames maps each doc comment to the names its declaration declares: a func's, a type's, a
// var's or const's, a field's or a method's.
func docNames(f *ast.File) map[*ast.CommentGroup][]string {
	names := map[*ast.CommentGroup][]string{}
	add := func(doc *ast.CommentGroup, idents ...*ast.Ident) {
		for _, ident := range idents {
			if doc != nil && ident != nil {
				names[doc] = append(names[doc], ident.Name)
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncDecl:
			add(n.Doc, n.Name)
		case *ast.GenDecl:
			for _, spec := range n.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					add(n.Doc, spec.Name)
					add(spec.Doc, spec.Name)
				case *ast.ValueSpec:
					add(n.Doc, spec.Names...)
					add(spec.Doc, spec.Names...)
				}
			}
		case *ast.Field:
			add(n.Doc, n.Names...)
		}
		return true
	})
	return names
}

// masked returns the body with x in place of each rune of a declared name that starts it, after
// A, An or The where one leads. Vale reads no word there then, as the name is code, and each
// column keeps its source's.
func masked(body string, names []string) string {
	for _, article := range []string{"", "A ", "An ", "The "} {
		rest, ok := strings.CutPrefix(body, article)
		for _, name := range names {
			after, found := strings.CutPrefix(rest, name)
			next, _ := utf8.DecodeRuneInString(after)
			if ok && found && next != '_' && !unicode.IsLetter(next) && !unicode.IsDigit(next) {
				return article + strings.Repeat("x", utf8.RuneCountInString(name)) + after
			}
		}
	}
	return body
}

// lineMirror mirrors the comments that marker starts, of full lines alone: a line rule cannot tell
// a trailing comment from a marker in a string.
func lineMirror(src []byte, marker string, directive *regexp.Regexp) *mirror {
	m := newMirror()
	for i, line := range strings.Split(string(src), "\n") {
		rest := strings.TrimLeft(line, " \t")
		body, found := strings.CutPrefix(rest, marker)
		body, skip, ok := text(body, directive, marker)
		if !found || !ok || i == 0 && strings.HasPrefix(line, "#!") {
			m.gap()
			continue
		}
		indent := utf8.RuneCountInString(line[:len(line)-len(rest)])
		m.add(body, origin{i + 1, indent + len(marker) + skip + 1})
	}
	return m
}

// unchecked matches Markdown, which Vale reads itself, and property lists. It matches a file not
// written here too: foundryctl's, Vale's styles, the toolkit's, pip-compile's, Go's, npm's, a license.
var unchecked = regexp.MustCompile(`^(?:\.vale/styles/|roles/server/ubuntu/signoz/(?:deployment/|casting\.yaml\.lock$)|` +
	`\.github/(?:ISSUE_TEMPLATE/|lint-requirements\.txt$)|go\.sum$)|(?:^|/)(?:LICENSE|package-lock\.json)$|\.(?:md|plist)$`)

// mirrorOf mirrors the file's comments by its kind: Go's, JSONC's and go.mod's, else those of #.
func mirrorOf(file string, src []byte) (*mirror, error) {
	base := path.Base(file)
	switch {
	case strings.HasSuffix(base, ".go"):
		return goMirror(src)
	case strings.HasSuffix(base, ".json"), strings.HasSuffix(base, ".json.tmpl"), base == "go.mod":
		return lineMirror(src, "//", nil), nil
	}
	return lineMirror(src, "#", hashDirective), nil
}
