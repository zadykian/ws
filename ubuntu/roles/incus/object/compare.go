package object

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
)

// differs returns the paths of the declared keys that have lacks or holds another value of. A map
// in want is compared by its keys alone, so that what Incus adds to an object differs from nothing.
// A list is compared whole, element by element. A declared empty list or map matches a key that
// have lacks or holds as null, as Incus leaves an empty one out or says null. A nil declares a key
// absent: it differs where have holds the key. path is the key's place, empty at the top.
func differs(want, have any, path string) []string {
	switch w := want.(type) {
	case map[string]any:
		h, ok := have.(map[string]any)
		if have == nil && len(w) == 0 {
			return nil
		}
		if !ok {
			return []string{cmpPath(path, "")}
		}
		var keys []string
		for _, k := range slices.Sorted(maps.Keys(w)) {
			there, found := h[k]
			switch {
			case found:
				keys = append(keys, differs(w[k], there, cmpPath(path, k))...)
			case !empty(w[k]):
				keys = append(keys, cmpPath(path, k))
			}
		}
		return keys
	case []any:
		h, ok := have.([]any)
		if have == nil && len(w) == 0 {
			return nil
		}
		if !ok || len(h) != len(w) {
			return []string{cmpPath(path, "")}
		}
		for i := range w {
			if len(differs(w[i], h[i], path)) > 0 {
				return []string{cmpPath(path, "")}
			}
		}
		return nil
	default:
		if text(want) != text(have) {
			return []string{cmpPath(path, "")}
		}
		return nil
	}
}

// cmpPath joins a key to the path of its map.
func cmpPath(path, key string) string {
	switch {
	case path == "":
		return key
	case key == "":
		return path
	}
	return path + "." + key
}

// empty reports whether a declared value is nil, an empty list or an empty map.
func empty(v any) bool {
	switch v := v.(type) {
	case nil:
		return true
	case map[string]any:
		return len(v) == 0
	case []any:
		return len(v) == 0
	}
	return false
}

// text is a scalar as its YAML or JSON says it, so that a number and its text compare equal.
func text(v any) string {
	if f, ok := v.(float64); ok {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}

// merge returns a copy of base with over laid on it: maps merge key by key, and any other value of
// over replaces base's. A nil of over removes the key.
func merge(base, over map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(over))
	maps.Copy(out, base)
	for k, v := range over {
		switch v := v.(type) {
		case nil:
			delete(out, k)
		case map[string]any:
			b, _ := out[k].(map[string]any)
			out[k] = merge(b, v)
		default:
			out[k] = v
		}
	}
	return out
}

// pick returns the entries of m under keys.
func pick(m map[string]any, keys []string) map[string]any {
	out := map[string]any{}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			out[k] = v
		}
	}
	return out
}

// plain turns the numbers that a JSON decode kept as text into int64 or float64, which YAML writes
// as numbers.
func plain(v any) any {
	switch v := v.(type) {
	case map[string]any:
		for k, e := range v {
			v[k] = plain(e)
		}
	case []any:
		for i, e := range v {
			v[i] = plain(e)
		}
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return n
		}
		if f, err := v.Float64(); err == nil {
			return f
		}
	}
	return v
}
