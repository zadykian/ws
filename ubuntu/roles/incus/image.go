package incus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/fetch"
	"github.com/zadykian/ws/engine/core/resource/file"
)

// imageSHA256 is the SHA-256 of Ubuntu's root tarball for each architecture, as the SHA256SUMS of
// the dated release lists it:
// https://cloud-images.ubuntu.com/releases/26.04/release-20261108/SHA256SUMS.
var imageSHA256 = map[string]string{
	"amd64": "daf9ae2d3f458072f7192839be1befb9b261def6ab69e75b47df0a44d468113d",
	"arm64": "766fbd0896c0dfb871100b58dfd78470d99cfa2066400557f2256c625144f268",
}

// imageURL is the root tarball of the pinned release, which carries cloud-init. The release names
// a dated directory, not the moving one, so that the SHA-256 holds.
func imageURL(release, arch string) string {
	return fmt.Sprintf("https://cloud-images.ubuntu.com/releases/%s/release-%s/ubuntu-%s-server-cloudimg-%s-root.tar.xz",
		imageVersion, release, imageVersion, arch)
}

// ImageAlias names the image that the guests launch from. It holds the release and the identity of
// the metadata, so that a new pin or a changed template is another image, and a guest keeps its own.
func ImageAlias(id string) string {
	return fmt.Sprintf("ws-ubuntu-%s-%s-%s", imageVersion, imageRelease, id)
}

// Image is Ubuntu's root tarball of the pinned release, imported with the metadata that ws writes.
// Once Incus holds the alias, nothing is due. Otherwise it downloads the tarball against its
// SHA-256, writes the metadata, imports both and drops the files.
type Image struct {
	M effect.Machine
	// sums replaces imageSHA256 where set, for a test whose page is not the release.
	sums map[string]string
}

// Alias returns the image's alias for this machine, which the architecture and the metadata give.
func (i Image) Alias(ctx context.Context) (string, error) {
	_, alias, err := i.metadata(ctx)
	return alias, err
}

// metadata builds the metadata for the machine's architecture, and returns it with the alias.
func (i Image) metadata(ctx context.Context) (tarball []byte, alias string, err error) {
	arch, err := fetch.Architecture(ctx, i.M)
	if err != nil {
		return nil, "", err
	}
	tarball, id, err := Metadata(arch)
	return tarball, ImageAlias(id), err
}

// Plan plans the import where Incus holds no image of the alias.
func (i Image) Plan(ctx context.Context) ([]resource.Change, error) {
	tarball, alias, err := i.metadata(ctx)
	if err != nil {
		return nil, err
	}
	held, err := i.held(ctx, alias)
	if err != nil || held {
		return nil, err
	}
	sums := i.sums
	if sums == nil {
		sums = imageSHA256
	}
	root, err := fetch.PinnedFile{M: i.M, Version: imageRelease, URL: imageURL, SHA256: sums}.Cached(ctx)
	if err != nil {
		return nil, err
	}
	meta := path.Join(fetch.CacheDir, alias+"-metadata.tar.gz")
	var changes []resource.Change
	for _, r := range []resource.Resource{
		file.Dir{M: i.M, Path: fetch.CacheDir, Mode: 0o755},
		root,
		file.File{M: i.M, Path: meta, Content: tarball, Mode: 0o600, Secret: true},
	} {
		more, err := r.Plan(ctx)
		if err != nil {
			return nil, err
		}
		changes = append(changes, more...)
	}
	return append(changes, importImage{m: i.M, alias: alias, files: [2]string{meta, root.Path}}), nil
}

// imageInfo is what incus image list prints of an image that ws reads.
type imageInfo struct {
	Aliases []struct {
		Name string `json:"name"`
	} `json:"aliases"`
}

// held reports whether Incus has an image under the alias.
func (i Image) held(ctx context.Context, alias string) (bool, error) {
	out, err := i.M.Run(ctx, "incus", "image", "list", "--format", "json")
	if err != nil {
		return false, err
	}
	var images []imageInfo
	if err := json.NewDecoder(bytes.NewReader(out)).Decode(&images); err != nil {
		return false, fmt.Errorf("incus image list: its output is no JSON array of images: %w", err)
	}
	return slices.ContainsFunc(images, func(image imageInfo) bool {
		return slices.ContainsFunc(image.Aliases, func(a struct {
			Name string `json:"name"`
		}) bool {
			return a.Name == alias
		})
	}), nil
}

// importImage imports the metadata and the rootfs under the alias, and drops both files.
type importImage struct {
	m     effect.Machine
	alias string
	// files are the metadata's and the rootfs's.
	files [2]string
}

func (c importImage) String() string {
	return fmt.Sprintf("import the image %s from %s and %s", c.alias, c.files[0], c.files[1])
}

func (c importImage) Apply(ctx context.Context) error {
	if _, err := c.m.Run(ctx, "incus", "image", "import", c.files[0], c.files[1], "--alias", c.alias); err != nil {
		return err
	}
	_, err := c.m.RunRoot(ctx, "rm", "-f", "--", c.files[0], c.files[1])
	return err
}
