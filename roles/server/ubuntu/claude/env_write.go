package claude

import (
	"encoding/json/jsontext"
	"sort"
)

// envEdit is what mergeEnv found to change in env.
type envEdit struct {
	// Set lists the keys whose value is new or differs, as KEY=VALUE: the values are the role's own.
	Set []string
	// Removed lists the keys removed, by name alone.
	Removed []string
}

func (e envEdit) none() bool { return len(e.Set) == 0 && len(e.Removed) == 0 }

// writeMissing adds the keys of set that env lacks, in the order of their name.
func writeMissing(enc *jsontext.Encoder, seen map[string]bool, set map[string]string, edit *envEdit) error {
	var names []string
	for k := range set {
		if !seen[k] {
			names = append(names, k)
		}
	}
	sort.Strings(names)
	for _, k := range names {
		if err := enc.WriteToken(jsontext.String(k)); err != nil {
			return err
		}
		if err := enc.WriteToken(jsontext.String(set[k])); err != nil {
			return err
		}
		edit.Set = append(edit.Set, k+"="+set[k])
	}
	return nil
}

// writeObject writes a whole env object that the settings lacked.
func writeObject(enc *jsontext.Encoder, seen map[string]bool, set map[string]string, edit *envEdit) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	if err := writeMissing(enc, seen, set, edit); err != nil {
		return err
	}
	return enc.WriteToken(jsontext.EndObject)
}
