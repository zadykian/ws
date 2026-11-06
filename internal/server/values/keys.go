package values

import (
	"reflect"
	"slices"
	"strings"
)

// Kind is what a key's value is.
type Kind int

// The kinds of a key that a line holds.
const (
	Text Kind = iota
	Number
	Decimal
	Switch
	List
)

// Key is a key of the file that a line of text holds: a scalar, or a list of strings. A table, such
// as fans_curves, stays the file's to edit.
type Key struct {
	Name string
	Kind Kind
	// place leads to the key's field in Values.
	place []int
}

// Keys lists the keys that a line holds, in the order of Values. They come from its fields, so that a
// role's new key needs no second list.
func Keys() []Key {
	return fieldKeys(reflect.TypeFor[Values](), nil)
}

// fieldKeys lists the keys of a struct at place, and of the structs inlined in it.
func fieldKeys(t reflect.Type, place []int) []Key {
	var keys []Key
	for i := range t.NumField() {
		field := t.Field(i)
		at := append(slices.Clone(place), i)
		name, options, _ := strings.Cut(field.Tag.Get("yaml"), ",")
		if options == "inline" {
			keys = append(keys, fieldKeys(field.Type, at)...)
		} else if kind, ok := kindOf(field.Type); ok {
			keys = append(keys, Key{name, kind, at})
		}
	}
	return keys
}

// kindOf is the kind of a field, and false for a table. A pointer is a key that may stay unset.
func kindOf(t reflect.Type) (Kind, bool) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return Text, true
	case reflect.Int:
		return Number, true
	case reflect.Float64:
		return Decimal, true
	case reflect.Bool:
		return Switch, true
	case reflect.Slice:
		return List, t.Elem().Kind() == reflect.String
	}
	return 0, false
}
