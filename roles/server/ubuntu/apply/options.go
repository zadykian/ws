// Package apply runs the server's roles for ws server ubuntu apply, each as Go steps.
package apply

// Options are the flags of ws server ubuntu apply.
type Options struct {
	// Check plans the steps and applies none.
	Check bool
	// Only names what runs, each by ROLE or ROLE/STEP, or is empty for all. The other steps of a
	// role named by ROLE/STEP alone are planned and left out, as later steps read their state.
	Only []string
	// Secrets is the age-encrypted file of --secrets, or empty.
	Secrets string
}
