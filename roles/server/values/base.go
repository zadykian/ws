package values

import "fmt"

// maxSwappiness is the kernel's highest vm.swappiness.
const maxSwappiness = 200

// check refuses a swap size or a swappiness that the kernel cannot take. base's units take each
// unchanged, so a wrong one fails the load, before any unit changes the swap.
func (b Base) check() error {
	if b.SwapSizeMB != nil && *b.SwapSizeMB < 0 {
		return fmt.Errorf("WS_BASE_SWAP_SIZE_MB is %d: it takes a whole number of MiB, or 0 for none", *b.SwapSizeMB)
	}
	if b.Swappiness < 0 || b.Swappiness > maxSwappiness {
		return fmt.Errorf("WS_BASE_SWAPPINESS is %d: it takes a whole number from 0 to %d", b.Swappiness, maxSwappiness)
	}
	return nil
}
