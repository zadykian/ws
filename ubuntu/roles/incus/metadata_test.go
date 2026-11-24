package incus

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"path"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/zadykian/ws/ubuntu/values"
)

// unpack returns the files of a metadata tarball, by name.
func unpack(t *testing.T, tarball []byte) map[string]string {
	zip, err := gzip.NewReader(bytes.NewReader(tarball))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for r := tar.NewReader(zip); ; {
		head, err := r.Next()
		if errors.Is(err, io.EOF) {
			return files
		}
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		files[head.Name] = string(body)
	}
}

func TestMetadataDescribesAContainerImageWithTheSeedsTemplates(t *testing.T) {
	tarball, _, err := Metadata("amd64")
	if err != nil {
		t.Fatal(err)
	}
	files := unpack(t, tarball)
	var meta struct {
		Architecture string `yaml:"architecture"`
		Templates    map[string]struct {
			When     []string `yaml:"when"`
			Template string   `yaml:"template"`
		} `yaml:"templates"`
	}
	if err := yaml.Unmarshal([]byte(files["metadata.yaml"]), &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Architecture != "x86_64" || len(meta.Templates) != 5 {
		t.Fatalf("metadata = %+v, want x86_64, the four seed files and the socket's directory", meta)
	}
	for _, name := range []string{"meta-data", "user-data", "vendor-data", "network-config"} {
		entry, ok := meta.Templates["/var/lib/cloud/seed/nocloud-net/"+name]
		if !ok || strings.Join(entry.When, ",") != "create,copy" || entry.Template != name+".tpl" {
			t.Errorf("template of %s = %+v, want it written as the instance is created", name, entry)
		}
		if _, ok := files["templates/"+name+".tpl"]; !ok {
			t.Errorf("the tarball lacks templates/%s.tpl", name)
		}
	}
	if got := files["templates/meta-data.tpl"]; !strings.Contains(got, "instance-id: {{ instance.name }}") || !strings.Contains(got, "local-hostname: {{ instance.name }}") {
		t.Errorf("meta-data template = %q, want the instance's name as its id and hostname", got)
	}
	if got := files["templates/network-config.tpl"]; !strings.Contains(got, `config_get("cloud-init.network-config"`) {
		t.Errorf("network-config template = %q, want the instance's key", got)
	}
}

// The proxy device makes the guest's socket and no directory, and a missing one fails the guest's
// start. So the image holds the directory, by a file that a template of Incus's makes.
func TestMetadataHoldsTheDirectoryOfTheOTLPSocket(t *testing.T) {
	tarball, _, err := Metadata("amd64")
	if err != nil {
		t.Fatal(err)
	}
	files := unpack(t, tarball)
	var meta struct {
		Templates map[string]struct {
			When     []string `yaml:"when"`
			Template string   `yaml:"template"`
		} `yaml:"templates"`
	}
	if err := yaml.Unmarshal([]byte(files["metadata.yaml"]), &meta); err != nil {
		t.Fatal(err)
	}
	entry, ok := meta.Templates[path.Dir(values.OTLPSocket)+"/README"]
	if !ok || strings.Join(entry.When, ",") != "create,copy" {
		t.Fatalf("templates = %+v, want a file in %s, written as the instance is created", meta.Templates, path.Dir(values.OTLPSocket))
	}
	if _, ok := files["templates/"+entry.Template]; !ok {
		t.Errorf("the tarball lacks templates/%s", entry.Template)
	}
	if strings.HasPrefix(values.OTLPSocket, "/run/") || strings.HasPrefix(values.OTLPSocket, "/tmp/") {
		t.Errorf("the socket %s is in /run or /tmp, which a guest mounts after its devices start", values.OTLPSocket)
	}
}

func TestMetadataIsTheSameEachTime(t *testing.T) {
	a, idA, err := Metadata("arm64")
	if err != nil {
		t.Fatal(err)
	}
	b, idB, _ := Metadata("arm64")
	if !bytes.Equal(a, b) || idA != idB || len(idA) != 8 {
		t.Errorf("two builds differ, or the identity %q is not 8 hex digits", idA)
	}
	if _, idC, _ := Metadata("amd64"); idC == idA {
		t.Errorf("amd64 and arm64 share the identity %s", idA)
	}
}

func TestMetadataRefusesAnArchitectureNoImageIsPinnedFor(t *testing.T) {
	if _, _, err := Metadata("riscv64"); err == nil || !strings.Contains(err.Error(), "riscv64") {
		t.Errorf("Metadata(riscv64) = %v, want it refused by name", err)
	}
}
