package host

// FixBackupName makes the backups' names fixed, and returns the function that restores them.
func FixBackupName(name string) func() {
	old := backupName
	backupName = func(path string) string { return path + "." + name + "~" }
	return func() { backupName = old }
}
