package effect

import (
	"bytes"
	"context"
	"io/fs"
	"path/filepath"
	"strconv"
)

// sudoWrite fills a new file beside path through sudo, gives it owner, as uid:gid, and perm, and
// renames it over path.
func (m OSMachine) sudoWrite(path string, data []byte, owner string, perm fs.FileMode) error {
	if err := m.ticket(true); err != nil {
		return err
	}
	tmp := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".ws-new")
	mode := strconv.FormatUint(uint64(perm.Perm()), 8)
	cmd := m.command(context.Background(),
		append(m.sudo(), "sh", "-c", rootWrite, "sh", tmp, owner, mode, path))
	cmd.Stdin = bytes.NewReader(data)
	if _, err := cmd.Output(); err != nil {
		_, _ = m.RunRoot(context.Background(), "rm", "-f", tmp)
		return commandError(cmd, err)
	}
	return nil
}
