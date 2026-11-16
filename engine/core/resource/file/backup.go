package file

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

// backupName names the copy of a file that Backup keeps: the path, the process and the time, then
// a tilde, as Ansible's backup does. A variable, so that a test fixes it.
var backupName = func(path string) string {
	return fmt.Sprintf("%s.%d.%s~", path, os.Getpid(), time.Now().Format("2006-01-02@15:04:05"))
}

// backup copies the file to its backup name, with its owner and mode, where the file exists and
// its content is not the new one. One read gives the copy its content, owner and mode. So a link
// swapped in after the guard is never copied, and a secret stays in the copy's mode.
func (f File) backup(content []byte) error {
	before, a, err := f.read()
	if errors.Is(err, fs.ErrNotExist) || (err == nil && bytes.Equal(before, content)) {
		return nil
	}
	if err != nil {
		return err
	}
	return f.M.WriteAtomic(backupName(f.Path), before, a)
}
