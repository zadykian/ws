package values

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
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

// String says what the kind takes, in a form's help and in a refusal.
func (k Kind) String() string {
	return [...]string{"text", "a whole number", "a number", "true or false", "a list, split by commas"}[k]
}

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

// Check says why the key can't take value. Each key takes an empty value, which gives its default
// back. It is pure, as a form runs it on each edit.
func (k Key) Check(value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return errors.New(k.Name + " takes one line")
	}
	if value == "" {
		return nil
	}
	var err error
	switch k.Kind {
	case Number:
		_, err = strconv.Atoi(value)
	case Decimal:
		var number float64
		number, err = strconv.ParseFloat(value, 64)
		if math.IsNaN(number) || math.IsInf(number, 0) {
			err = strconv.ErrSyntax
		}
	case Switch:
		if value != "true" && value != "false" {
			err = strconv.ErrSyntax
		}
	case Text, List:
	}
	if err != nil {
		return fmt.Errorf("%s takes %s", k.Name, k.Kind)
	}
	return nil
}

// Text is the key's value in v as a line gives it, which is empty for an unset pointer.
func (v Values) Text(k Key) string {
	field := reflect.ValueOf(v).FieldByIndex(k.place)
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return ""
		}
		field = field.Elem()
	}
	if k.Kind == List {
		return strings.Join(field.Interface().([]string), ", ")
	}
	return fmt.Sprint(field.Interface())
}
