package incus

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
)

const imageList = "incus image list --format json"

// imageMachine returns a machine of amd64 whose page is body, with Image's sums set to its own.
func imageMachine(t *testing.T, body string, listed string) (*effecttest.Machine, Image, string) {
	t.Helper()
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results[imageList] = effecttest.Result{Out: []byte(listed)}
	m.Pages[imageURL(imageRelease, "amd64")] = []byte(body)
	image := Image{M: m, sums: map[string]string{"amd64": fmt.Sprintf("%x", sha256.Sum256([]byte(body)))}}
	alias, err := image.Alias(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return m, image, alias
}

func TestTheImageIsPinnedByASumForEachArchitectureItRuns(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		if len(imageSHA256[arch]) != 64 {
			t.Errorf("the pin for %s is %q, not a SHA-256", arch, imageSHA256[arch])
		}
		if url := imageURL(imageRelease, arch); !strings.Contains(url, "/release-"+imageRelease+"/") || !strings.HasSuffix(url, arch+"-root.tar.xz") {
			t.Errorf("URL for %s = %s, want the dated release's root tarball", arch, url)
		}
	}
}

func TestAnImageUnderTheAliasIsLeftAlone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	alias, err := Image{M: m}.Alias(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	m.Results[imageList] = effecttest.Result{Out: []byte(`[{"fingerprint":"a","aliases":[]},{"fingerprint":"b","aliases":[{"name":"x"},{"name":"` + alias + `"}]}]`)}
	if changes, err := (Image{M: m}).Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("changes = %v, err = %v, want none, and no download", changes, err)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "download") }) {
		t.Errorf("calls = %q, want no download", m.Calls)
	}
}

func TestAMissingImageIsDownloadedImportedAndDropped(t *testing.T) {
	m, image, alias := imageMachine(t, "rootfs", "[]")
	m.Results["incus image import /var/cache/ws/"+alias+"-metadata.tar.gz /var/cache/ws/ubuntu-26.04-server-cloudimg-amd64-root.tar.xz --alias "+alias] = effecttest.Result{}
	changes, err := image.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var heads []string
	for _, c := range changes {
		heads = append(heads, strings.SplitN(c.String(), "\n", 2)[0])
		if err := c.Apply(t.Context()); err != nil {
			t.Fatalf("%s: %v", c, err)
		}
	}
	want := []string{
		"create directory /var/cache/ws", "download " + imageURL(imageRelease, "amd64"),
		"create /var/cache/ws/" + alias + "-metadata.tar.gz", "import the image " + alias,
	}
	if len(heads) != len(want) {
		t.Fatalf("changes = %q, want the directory, the download, the metadata and the import", heads)
	}
	for i := range want {
		if !strings.HasPrefix(heads[i], want[i]) {
			t.Errorf("change %d = %q, want it to start %q", i, heads[i], want[i])
		}
	}
	if left := slices.Sorted(maps.Keys(m.Files)); len(left) != 0 {
		t.Errorf("the cache keeps %q after the import", left)
	}
}

func TestAnImageWhoseDownloadFailsItsSumIsNeverImported(t *testing.T) {
	m, image, _ := imageMachine(t, "rootfs", "[]")
	m.Pages[imageURL(imageRelease, "amd64")] = []byte("another body")
	changes, err := image.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var failed error
	for _, c := range changes {
		if failed = c.Apply(t.Context()); failed != nil {
			break
		}
	}
	if failed == nil || !strings.Contains(failed.Error(), "SHA-256") {
		t.Fatalf("applying = %v, want the sum refused", failed)
	}
	for _, call := range m.Calls {
		if strings.Contains(call, "image import") {
			t.Errorf("calls %q: the import ran on a body that missed its sum", m.Calls)
		}
	}
}

func TestAnImageOfAnArchitectureWithNoPinIsRefused(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("s390x\n")}
	m.Results[imageList] = effecttest.Result{Out: []byte("[]")}
	if _, err := (Image{M: m}).Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "s390x") {
		t.Errorf("plan = %v, want s390x named", err)
	}
}

var _ resource.Resource = Image{}
