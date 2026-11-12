package host

import "time"

// FixBackupName makes the backups' names fixed, and returns the function that restores them.
func FixBackupName(name string) func() {
	old := backupName
	backupName = func(path string) string { return path + "." + name + "~" }
	return func() { backupName = old }
}

// FixRetryWait sets the wait between a fetch's tries, and returns the function that restores it.
func FixRetryWait(wait time.Duration) func() {
	old := retryWait
	retryWait = wait
	return func() { retryWait = old }
}
