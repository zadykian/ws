package settings

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
)

// errNotSettings is the error of a file that is not a JSON object whose env is an object.
var errNotSettings = errors.New("is not a JSON object whose env is an object")

// mergeEnv returns the settings with the env keys of set added or replaced and those of removed
// gone. Every other key keeps its place, and a new one goes at the end, in the order of its name.
// The result is written as claude writes the file: two spaces, a member a line, non-ASCII as it
// is, a newline at the end. Empty data holds no settings yet.
func mergeEnv(data []byte, set map[string]string, removed []string) ([]byte, envEdit, error) {
	var edit envEdit
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		data = []byte("{}")
	}
	dec := jsontext.NewDecoder(bytes.NewReader(data))
	var out bytes.Buffer
	enc := jsontext.NewEncoder(&out, jsontext.WithIndent("  "))
	if k := dec.PeekKind(); k != '{' {
		return nil, edit, errNotSettings
	}
	if err := copyToken(dec, enc); err != nil {
		return nil, edit, err
	}
	sawEnv := false
	for dec.PeekKind() != '}' {
		name, err := dec.ReadToken()
		if err != nil {
			return nil, edit, err
		}
		if err := enc.WriteToken(name); err != nil {
			return nil, edit, err
		}
		if name.String() != "env" {
			if err := copyValue(dec, enc); err != nil {
				return nil, edit, err
			}
			continue
		}
		sawEnv = true
		if dec.PeekKind() != '{' {
			return nil, edit, errNotSettings
		}
		if edit, err = mergeObject(dec, enc, set, removed); err != nil {
			return nil, edit, err
		}
	}
	if !sawEnv {
		if err := enc.WriteToken(jsontext.String("env")); err != nil {
			return nil, edit, err
		}
		if err := writeObject(enc, nil, set, &edit); err != nil {
			return nil, edit, err
		}
	}
	if err := copyToken(dec, enc); err != nil {
		return nil, edit, err
	}
	if _, err := dec.ReadToken(); !errors.Is(err, io.EOF) {
		return nil, edit, fmt.Errorf("text after the JSON object: %w", err)
	}
	return append(bytes.TrimRight(out.Bytes(), "\n"), '\n'), edit, nil
}
