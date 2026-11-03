// Package apply runs the server's roles for ws server apply, each as Go steps.
package apply

// Options are the flags of ws server apply.
type Options struct {
	// Check plans the steps and applies none.
	Check bool
	// Only names the roles to run, or is empty for all.
	Only []string
	// Secrets is the age-encrypted file of --secrets, or empty.
	Secrets string
}
