package settings

import (
	"encoding/json"
	"encoding/json/jsontext"
	"slices"
)

func copyToken(dec *jsontext.Decoder, enc *jsontext.Encoder) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	return enc.WriteToken(tok)
}

func copyValue(dec *jsontext.Decoder, enc *jsontext.Encoder) error {
	v, err := dec.ReadValue()
	if err != nil {
		return err
	}
	return enc.WriteValue(v)
}

// mergeObject copies the env object, replacing and dropping members.
func mergeObject(dec *jsontext.Decoder, enc *jsontext.Encoder, set map[string]string, removed []string) (envEdit, error) {
	var edit envEdit
	if err := copyToken(dec, enc); err != nil {
		return edit, err
	}
	seen := map[string]bool{}
	for dec.PeekKind() != '}' {
		name, err := dec.ReadToken()
		if err != nil {
			return edit, err
		}
		key := name.String()
		value, err := dec.ReadValue()
		if err != nil {
			return edit, err
		}
		seen[key] = true
		switch want, ok := set[key]; {
		case slices.Contains(removed, key) && !ok:
			edit.Removed = append(edit.Removed, key)
			continue
		case ok:
			quoted, err := jsontext.AppendQuote(nil, want)
			if err != nil {
				return edit, err
			}
			var old string
			if json.Unmarshal(value, &old) != nil || old != want {
				edit.Set = append(edit.Set, key+"="+want)
			}
			value = jsontext.Value(quoted)
		}
		if err := enc.WriteToken(jsontext.String(key)); err != nil {
			return edit, err
		}
		if err := enc.WriteValue(value); err != nil {
			return edit, err
		}
	}
	if err := writeMissing(enc, seen, set, &edit); err != nil {
		return edit, err
	}
	return edit, copyToken(dec, enc)
}
