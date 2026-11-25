package incus

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/zadykian/ws/ubuntu/values"
)

// The release of Ubuntu's cloud images that ws pins, by its version and its date. A newer release
// is a new pin: its SHA-256 for each architecture, in image.go, and this date.
const (
	imageVersion = "26.04"
	imageRelease = "20261108"
)

// imageFiles holds the templates of the image's metadata. The image holds no host key, no name of
// its own and an empty netplan, so they write cloud-init's seed, which makes them on the first boot.
//
//go:embed files/image
var imageFiles embed.FS

// seedDir is where cloud-init reads its NoCloud seed.
const seedDir = "/var/lib/cloud/seed/nocloud-net"

// cloudConfig is the user-data and the vendor-data of a guest that sets none.
const cloudConfig = "#cloud-config\n{}"

// seeds lists the seed's files. A template with a key reads it from the instance's config, and
// takes the default where the instance sets none. meta-data takes the instance's name alone.
var seeds = []struct{ file, key, fallback string }{
	{file: "meta-data"},
	{file: "user-data", key: "cloud-init.user-data", fallback: cloudConfig},
	{file: "vendor-data", key: "cloud-init.vendor-data", fallback: cloudConfig},
	{file: "network-config", key: "cloud-init.network-config", fallback: "version: 2\nethernets:\n  eth0:\n    dhcp4: true"},
}

// socketMark is the one file of the directory of the OTLP socket. The profile's proxy device makes
// the socket, never its directory, and a missing one fails the guest's start. Incus makes a
// template's directory, so the image holds it by this file. The directory stays outside /run and
// /tmp, which a guest mounts after its devices start.
const socketMark = "otlp.tpl"

// incusArch names the architectures of an image as Incus does, by dpkg's name.
var incusArch = map[string]string{"amd64": "x86_64", "arm64": "aarch64"}

// Metadata returns the tarball of an image's metadata for the architecture, which dpkg names, and
// an identity for it: the digest of what it holds. Incus takes the tarball with the rootfs, and
// the identity names the image, so that a changed template makes another image. The tarball
// is the same for the same input, as its entries are in order and carry no time of the build.
func Metadata(arch string) (tarball []byte, id string, err error) {
	incus, ok := incusArch[arch]
	if !ok {
		return nil, "", fmt.Errorf("ws sets up guests on x86_64 and aarch64, not on %s", arch)
	}
	templates := map[string]any{}
	files := map[string][]byte{}
	for _, s := range seeds {
		body, err := fs.ReadFile(imageFiles, path.Join("files/image", s.file+".tpl"))
		if err != nil {
			return nil, "", err
		}
		files[path.Join("templates", s.file+".tpl")] = body
		entry := map[string]any{"when": []string{"create", "copy"}, "template": s.file + ".tpl"}
		if s.key != "" {
			entry["properties"] = map[string]any{"default": s.fallback}
		}
		templates[path.Join(seedDir, s.file)] = entry
	}
	mark, err := fs.ReadFile(imageFiles, path.Join("files/image", socketMark))
	if err != nil {
		return nil, "", err
	}
	files[path.Join("templates", socketMark)] = mark
	templates[path.Join(path.Dir(values.OTLPSocket), "README")] = map[string]any{"when": []string{"create", "copy"}, "template": socketMark}
	created := imageCreated()
	meta, err := yaml.Marshal(map[string]any{
		"architecture":  incus,
		"creation_date": created.Unix(),
		"properties": map[string]any{
			"description":  fmt.Sprintf("Ubuntu %s LTS, release %s, with ws's cloud-init seed", imageVersion, imageRelease),
			"os":           "Ubuntu",
			"release":      imageVersion,
			"serial":       imageRelease,
			"architecture": arch,
		},
		"templates": templates,
	})
	if err != nil {
		return nil, "", err
	}
	files["metadata.yaml"] = meta
	files["templates/"] = nil
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var buf bytes.Buffer
	zip := gzip.NewWriter(&buf)
	archive := tar.NewWriter(zip)
	sum := sha256.New()
	for _, name := range names {
		head := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(files[name])), ModTime: created, Typeflag: tar.TypeReg}
		if strings.HasSuffix(name, "/") {
			head.Mode, head.Typeflag = 0o755, tar.TypeDir
		}
		if err := archive.WriteHeader(head); err != nil {
			return nil, "", err
		}
		if _, err := archive.Write(files[name]); err != nil {
			return nil, "", err
		}
		_, _ = fmt.Fprintf(sum, "%s\x00%d\x00%s\x00", name, len(files[name]), files[name])
	}
	if err := archive.Close(); err != nil {
		return nil, "", err
	}
	if err := zip.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), fmt.Sprintf("%x", sum.Sum(nil))[:8], nil
}

// imageCreated is the day of the release, which the metadata dates the image by.
func imageCreated() time.Time {
	day, err := time.Parse("20060102", imageRelease)
	if err != nil {
		panic("imageRelease is no date: " + err.Error())
	}
	return day
}
