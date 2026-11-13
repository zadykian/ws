package cli

import (
	"github.com/zadykian/ws/engine/core/form"
	"github.com/zadykian/ws/engine/tui"
)

// FieldSpecs is a form's fields on the screen. A key that takes one of a few is a choice.
func FieldSpecs(fields []form.Field) []tui.FieldSpec {
	specs := make([]tui.FieldSpec, 0, len(fields))
	for _, f := range fields {
		spec := tui.FieldSpec{Label: f.Key, Help: f.Text, Value: f.Value, Default: f.Default,
			Options: f.Options, Valid: f.Valid}
		if f.Options != nil {
			spec.Kind = tui.ChoiceField
		}
		specs = append(specs, spec)
	}
	return specs
}
