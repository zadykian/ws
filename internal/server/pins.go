// Package server holds the plans of ws server: what the server's ws unpacks, installs and runs.
package server

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

// The names of the stages that report to the wizard's job.
const (
	packagesStage = "Packages"
	playbookStage = "Playbook"
)

// Pins takes the collections of a requirements.yml, as tools/collections reads them, and returns
// the file name of each one's tarball, NS-NAME-VERSION.tar.gz, sorted. A name with no version is
// an error.
func Pins(requirements []byte) ([]string, error) {
	var pins []string
	name := ""
	sc := bufio.NewScanner(bytes.NewReader(requirements))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if rest, ok := strings.CutPrefix(line, "- name:"); ok {
			if name != "" {
				return nil, fmt.Errorf("the collection %s has no version", name)
			}
			name = unquote(rest)
		} else if rest, ok := strings.CutPrefix(line, "version:"); ok {
			if name == "" {
				return nil, errors.New("a version has no collection")
			}
			pins = append(pins, strings.Replace(name, ".", "-", 1)+"-"+unquote(rest)+".tar.gz")
			name = ""
		}
	}
	if name != "" {
		return nil, fmt.Errorf("the collection %s has no version", name)
	}
	sort.Strings(pins)
	return pins, nil
}

// unquote trims the spaces and the quotes around a YAML scalar.
func unquote(s string) string {
	return strings.Trim(strings.TrimSpace(s), `"'`)
}

// Digest returns the hex SHA-256 of the regular files of fsys, in the order WalkDir visits them.
// For each file it takes the path, a NUL, the size in decimal, a NUL and the bytes.
func Digest(fsys fs.FS) (string, error) {
	h := sha256.New()
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		f, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }() // read-only
		info, err := f.Stat()
		if err != nil {
			return err
		}
		if _, err := io.WriteString(h, path+"\x00"+strconv.FormatInt(info.Size(), 10)+"\x00"); err != nil {
			return err
		}
		_, err = io.Copy(h, f)
		return err
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
