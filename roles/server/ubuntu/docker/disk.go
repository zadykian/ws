package docker

import "fmt"

// dockerRoot is where Docker keeps its data. Until dockerd first starts, the directory is missing
// unless a mount makes it, and the data will lie on the filesystem of its parent.
const (
	dockerRoot = "/var/lib/docker"
	dockerBase = "/var/lib"
)

// DiskSize returns the size in bytes of the filesystem that holds path. A path it cannot read
// fails.
type DiskSize func(path string) (uint64, error)

// dataDiskSize reads the size of the filesystem that holds Docker's data, as df did. The facts
// leave out a root whose device is no path, such as a container's overlay or a ZFS dataset.
func dataDiskSize(exists func(string) bool, size DiskSize) (uint64, error) {
	dir := dockerRoot
	if !exists(dir) {
		dir = dockerBase
	}
	n, err := size(dir)
	if err != nil {
		return 0, fmt.Errorf("size of the filesystem of %s: %w", dir, err)
	}
	return n, nil
}
