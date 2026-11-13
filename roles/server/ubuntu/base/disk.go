package base

// disk is what the swap step reads of the disk and changes on it, other than through the machine.
// The step runs as root, so these calls need no sudo. osDisk is the real one; tests use a fake.
type disk interface {
	// Statfs returns the MiB available to a non-root writer on the file system of path.
	Statfs(path string) (int64, error)
	// Stat describes the path without following a link; a missing path is not an error.
	Stat(path string) (swapFileInfo, error)
	// Allocate makes a file of mib MiB, mode 0600, owned by root. It fails where the file exists.
	Allocate(path string, mib int64) error
	// Restrict makes the file root's, mode 0600.
	Restrict(path string) error
	Rename(oldPath, newPath string) error
	// Remove removes the file, and a missing file is no error.
	Remove(path string) error
}
