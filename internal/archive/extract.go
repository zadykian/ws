// Package archive extracts tar archives into a directory.
package archive

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
)

// gzipMagic starts every gzip stream.
var gzipMagic = []byte{0x1f, 0x8b}

// Extract extracts a tar archive, gzip-compressed or plain, into root. It drops the first strip
// components of each name, such as the one directory that a release tarball holds, and skips an
// entry that has no more. It writes through root alone, so no entry leaves it, whatever its name
// or the links it made before. It handles directories, files, symlinks and hard links, and fails on
// any other entry.
func Extract(r io.Reader, root *os.Root, strip int) error {
	br := bufio.NewReader(r)
	if head, err := br.Peek(len(gzipMagic)); err == nil && string(head) == string(gzipMagic) {
		zr, err := gzip.NewReader(br)
		if err != nil {
			return err
		}
		defer func() { _ = zr.Close() }() // a read only stream
		r = zr
	} else {
		r = br
	}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := extract(tr, hdr, root, strip); err != nil {
			return fmt.Errorf("%s: %w", hdr.Name, err)
		}
	}
}

// stripped returns the name without its first strip components, and whether any are left.
func stripped(name string, strip int) (string, bool, error) {
	if path.IsAbs(name) {
		return "", false, errors.New("an absolute name")
	}
	parts := strings.Split(path.Clean(name), "/")
	if slices.Contains(parts, "..") {
		return "", false, errors.New("a name that leaves the directory")
	}
	if len(parts) <= strip {
		return "", false, nil
	}
	return path.Join(parts[strip:]...), true, nil
}

func extract(r io.Reader, hdr *tar.Header, root *os.Root, strip int) error {
	if hdr.Typeflag == tar.TypeXGlobalHeader {
		return nil
	}
	name, ok, err := stripped(hdr.Name, strip)
	if err != nil || !ok {
		return err
	}
	perm := fs.FileMode(uint32(hdr.Mode & 0o777))
	if dir := path.Dir(name); dir != "." {
		if err := root.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	switch hdr.Typeflag {
	case tar.TypeDir:
		if err := root.MkdirAll(name, 0o755); err != nil {
			return err
		}
		return root.Chmod(name, perm)
	case tar.TypeReg:
		f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, r)
		if err := errors.Join(err, f.Close()); err != nil {
			return err
		}
		return root.Chmod(name, perm)
	case tar.TypeSymlink:
		return root.Symlink(hdr.Linkname, name)
	case tar.TypeLink:
		target, ok, err := stripped(hdr.Linkname, strip)
		if err != nil || !ok {
			return errors.Join(err, errors.New("a hard link to a name outside the tree"))
		}
		return root.Link(target, name)
	default:
		return fmt.Errorf("an entry of type %q", hdr.Typeflag)
	}
}
