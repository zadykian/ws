package secrets

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// reference matches ${kind.name}; kinds and names are of letters, digits and underscores.
var reference = regexp.MustCompile(`^\$\{([a-z]+)\.([A-Za-z0-9_]+)\}`)

// Expand returns a copy of s with each reference replaced by the value it names, and $$ by $. A
// reference stands in the strings of env, git (token and username) and password, and names a
// token, an env or a password. A missing name, a loop and any other $ fail, naming kinds and names
// alone.
func Expand(s Secrets) (Secrets, error) {
	e := expander{s: s, done: map[string]string{}}
	out := s
	out.Env = nil
	out.Git = nil
	var err error
	for _, name := range slices.Sorted(maps.Keys(s.Env)) {
		var v string
		if v, err = e.value("env." + name); err != nil {
			return Secrets{}, err
		}
		if out.Env == nil {
			out.Env = map[string]string{}
		}
		out.Env[name] = v
	}
	for _, url := range slices.Sorted(maps.Keys(s.Git)) {
		g := s.Git[url]
		if g.Token, err = e.text("git."+url+".token", g.Token); err != nil {
			return Secrets{}, err
		}
		if g.Username, err = e.text("git."+url+".username", g.Username); err != nil {
			return Secrets{}, err
		}
		if out.Git == nil {
			out.Git = map[string]Git{}
		}
		out.Git[url] = g
	}
	if out.Password.VPNP12, err = e.value("password.vpn_p12"); err != nil {
		return Secrets{}, err
	}
	return out, nil
}

type expander struct {
	s     Secrets
	done  map[string]string
	stack []string
}

// raw returns the string that the reference "kind.name" names, as written, and whether it exists.
func (e *expander) raw(id string) (string, bool) {
	kind, name, _ := strings.Cut(id, ".")
	switch kind {
	case "token":
		switch name {
		case "github":
			return e.s.Token.GitHub, e.s.Token.GitHub != ""
		case "azure_devops":
			return e.s.Token.AzureDevOps, e.s.Token.AzureDevOps != ""
		}
	case "env":
		v, ok := e.s.Env[name]
		return v, ok
	case "password":
		if name == "vpn_p12" {
			return e.s.Password.VPNP12, e.s.Password.VPNP12 != ""
		}
	}
	return "", false
}

// value expands the string that id names, once.
func (e *expander) value(id string) (string, error) {
	if v, ok := e.done[id]; ok {
		return v, nil
	}
	if slices.Contains(e.stack, id) {
		return "", fmt.Errorf("%s refers to itself: %s", id, strings.Join(append(slices.Clone(e.stack), id), " -> "))
	}
	raw, _ := e.raw(id)
	e.stack = append(e.stack, id)
	v, err := e.text(id, raw)
	e.stack = e.stack[:len(e.stack)-1]
	if err != nil {
		return "", err
	}
	e.done[id] = v
	return v, nil
}

// text expands the references in a string that belongs to the key "where".
func (e *expander) text(where, raw string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		if raw[i] != '$' {
			b.WriteByte(raw[i])
			continue
		}
		rest := raw[i:]
		if strings.HasPrefix(rest, "$$") {
			b.WriteByte('$')
			i++
			continue
		}
		m := reference.FindStringSubmatch(rest)
		if m == nil {
			return "", fmt.Errorf("%s holds a $ that is no ${kind.name} and no $$", where)
		}
		id := m[1] + "." + m[2]
		if _, ok := e.raw(id); !ok {
			return "", fmt.Errorf("%s refers to %s, which is missing", where, id)
		}
		v, err := e.value(id)
		if err != nil {
			return "", err
		}
		b.WriteString(v)
		i += len(m[0]) - 1
	}
	return b.String(), nil
}
