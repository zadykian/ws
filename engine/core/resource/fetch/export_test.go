package fetch

import "time"

// FixRetryWait sets the wait between a fetch's tries, and returns the function that restores it.
func FixRetryWait(wait time.Duration) func() {
	old := retryWait
	retryWait = wait
	return func() { retryWait = old }
}
