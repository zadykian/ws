package values

import (
	"reflect"
	"strings"
)

// tags returns the yaml keys of the struct s, inlined structs included.
func tags(s reflect.Type) []string {
	var keys []string
	for i := range s.NumField() {
		f := s.Field(i)
		tag := f.Tag.Get("yaml")
		if strings.Contains(tag, "inline") {
			keys = append(keys, tags(f.Type)...)
			continue
		}
		keys = append(keys, strings.Split(tag, ",")[0])
	}
	return keys
}

func contains(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}

// fieldByTag returns the field of v whose yaml key is key, searching inlined structs.
func fieldByTag(v reflect.Value, key string) reflect.Value {
	for i := range v.NumField() {
		f := v.Type().Field(i)
		tag := f.Tag.Get("yaml")
		if strings.Contains(tag, "inline") {
			if got := fieldByTag(v.Field(i), key); got.IsValid() {
				return got
			}
			continue
		}
		if strings.Split(tag, ",")[0] == key {
			return v.Field(i)
		}
	}
	return reflect.Value{}
}

// normalize makes nil and empty collections, which YAML writes alike, compare equal.
func normalize(v any) any {
	switch x := v.(type) {
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = normalize(e)
		}
		return out
	case map[string]any:
		out := map[string]any{}
		for k, e := range x {
			out[k] = normalize(e)
		}
		return out
	}
	return v
}
