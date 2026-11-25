package keyboard_test

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"slices"
	"strings"
	"testing"
)

// icnsEntry is one image of an icns file: its type and its side in pixels.
type icnsEntry struct {
	Type string
	Side int
}

// icnsEntries are the images each icon holds: 16 and 32 pixels in ARGB, and their Retina twins,
// 32 and 64 pixels, in PNG.
var icnsEntries = []icnsEntry{{"ic04", 16}, {"ic05", 32}, {"ic11", 32}, {"ic12", 64}}

// TestTheIconsAreTemplates holds each layout's icon to its entries. Each is a square of its side,
// black but for its alpha, as a template icon is. Info.plist must mark both icons as templates.
func TestTheIconsAreTemplates(t *testing.T) {
	for _, name := range []string{"ENG", "RUS"} {
		data := roleFile(t, source+"Contents/Resources/"+name+".icns")
		if len(data) < 8 || string(data[:4]) != "icns" || int(binary.BigEndian.Uint32(data[4:8])) != len(data) {
			t.Fatalf("%s.icns: no icns header of its length", name)
		}
		var got []icnsEntry
		for rest := data[8:]; len(rest) > 0; {
			if len(rest) < 8 {
				t.Fatalf("%s.icns: an entry runs past the file", name)
			}
			n := int(binary.BigEndian.Uint32(rest[4:8]))
			if n < 8 || n > len(rest) {
				t.Fatalf("%s.icns: an entry runs past the file", name)
			}
			typ, body := string(rest[:4]), rest[8:n]
			rest = rest[n:]
			var side int
			if typ == "ic04" || typ == "ic05" {
				side = argbAlpha(t, name, typ, body)
			} else {
				side = pngAlpha(t, name, typ, body)
			}
			got = append(got, icnsEntry{typ, side})
		}
		if !slices.Equal(got, icnsEntries) {
			t.Errorf("%s.icns holds %v, want %v", name, got, icnsEntries)
		}
	}
	plist := string(roleFile(t, source+"Contents/Info.plist"))
	if n := strings.Count(plist, "<key>TISIconIsTemplate</key>\n\t\t<true/>"); n != 2 {
		t.Errorf("Info.plist marks %d icons as templates, want 2", n)
	}
}

// argbAlpha decodes an ARGB entry, whose planes, A, R, G and B, each run-length coded, follow
// "ARGB", and returns its side. Its colour planes must be black.
func argbAlpha(t *testing.T, name, typ string, body []byte) int {
	p, ok := bytes.CutPrefix(body, []byte("ARGB"))
	if !ok {
		t.Fatalf("%s.icns %s: no ARGB mark", name, typ)
	}
	// A byte below 0x80 says that so many bytes and one follow as they are; one from 0x80, that
	// the next byte repeats so many times less 0x80, and three.
	var out []byte
	for i := 0; i < len(p); {
		c := int(p[i])
		switch {
		case c < 0x80 && i+c+2 <= len(p):
			out = append(out, p[i+1:i+c+2]...)
			i += c + 2
		case c >= 0x80 && i+1 < len(p):
			out = append(out, bytes.Repeat(p[i+1:i+2], c-0x80+3)...)
			i += 2
		default:
			t.Fatalf("%s.icns %s: a run ends early", name, typ)
		}
	}
	side := 0
	for side*side*4 < len(out) {
		side++
	}
	if side*side*4 != len(out) || bytes.ContainsFunc(out[side*side:], func(r rune) bool { return r != 0 }) {
		t.Errorf("%s.icns %s: %d bytes, not a black square's four planes", name, typ, len(out))
	}
	return side
}

// pngAlpha decodes a PNG entry and returns its side. Its pixels must be black.
func pngAlpha(t *testing.T, name, typ string, body []byte) int {
	img, err := png.Decode(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("%s.icns %s: %v", name, typ, err)
	}
	b := img.Bounds()
	if b.Dx() != b.Dy() || b.Min != (image.Point{}) {
		t.Errorf("%s.icns %s: %v is not a square", name, typ, b)
	}
	for y := range b.Dy() {
		for x := range b.Dx() {
			if r, g, bl, _ := img.At(x, y).RGBA(); r|g|bl != 0 {
				t.Fatalf("%s.icns %s: pixel %d,%d is not black", name, typ, x, y)
			}
		}
	}
	return b.Dx()
}
