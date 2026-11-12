package fans

// The driver nct6687, which DKMS builds for the board's chip. These are a commit of
// github.com/Fred78290/nct6687d, which tags no releases, and the SHA-256 of GitHub's tarball of it.
// DKMS's version of the driver is the commit, short, as the driver's own dkms.conf has a fixed 1.
const (
	nct6687Commit = "7e3d7b36c55f06ab5bb2077bc85329f592996b2a"
	nct6687SHA256 = "d45306407920334b7a3e11c6a89c0527cbd5557b4efac343b96de7b4bffa850a"
	nct6687Short  = "7e3d7b36c55f"
)
