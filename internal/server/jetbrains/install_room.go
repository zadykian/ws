package jetbrains

import "fmt"

// room checks that Gateway's directory has 4 times the archive's size free, as JetBrains asks, for
// the archive and the build unpacked.
func (c install) room() error {
	have, err := c.s.Avail(c.s.Dist)
	if err != nil {
		return err
	}
	const mib = 1 << 20
	if need := uint64(max(c.b.Size, 0)) * 4; have < need {
		return fmt.Errorf("%s has %d MiB available, and %s needs 4 times its archive's %d MiB. "+
			"Its older builds there go only once this one is in place: free some, or leave %s out of "+
			"jetbrains_backends", c.s.Dist, have/mib, c.b.Name, c.b.Size/mib, c.b.Code)
	}
	return nil
}
