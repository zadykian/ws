package wizard

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

// Homebrew's release: its Homebrew.pkg, that file's SHA-256, and its signer, as pkgutil names it.
const (
	brewVersion = "7.0.10"
	brewURL     = "https://github.com/Homebrew/brew/releases/download/" + brewVersion + "/Homebrew.pkg"
	brewSHA256  = "ae13b8b0ac910730d3e982a9e7518288d17fd0ad272e42f412027e8ab4555575"
	brewSigner  = "Developer ID Installer: Patrick Linnane (927JGANW46)"
)

// brewPath is where Homebrew's .pkg puts brew on an Apple silicon Mac.
const brewPath = "/opt/homebrew/bin/brew"

// Homebrew installs Homebrew from its pinned .pkg, checked by its SHA-256 and its signer, through
// sudo.
type Homebrew struct {
	// Lock is the run's lock directory, which holds the download.
	Lock string
	// Sum is the .pkg's SHA-256, brewSHA256 where empty. Tests set their own.
	Sum string
	M   effect.Machine
	Out effect.Out
}

// Name is the stage's name.
func (*Homebrew) Name() string { return stage.Homebrew }

// Side is the Mac: Homebrew.
func (*Homebrew) Side() guided.Side { return stage.Mac }

// WaitsOn is Tools: the installer needs the Command Line Tools.
func (*Homebrew) WaitsOn() string { return stage.Tools }

// brewInstall is the change that installs Homebrew.
type brewInstall struct{}

func (brewInstall) String() string {
	return stage.Homebrew + ": would install Homebrew " + brewVersion + " from its Homebrew.pkg, through sudo"
}

func (h *Homebrew) done(ctx context.Context) bool {
	_, err := h.M.Run(ctx, brewPath, "--version")
	return err == nil
}

// Plan plans the install where brew does not run.
func (h *Homebrew) Plan(ctx context.Context) (step.Plan, error) {
	if h.done(ctx) {
		return step.Plan{}, nil
	}
	return step.Plan{Changes: []step.Change{brewInstall{}}}, nil
}

// Apply downloads the .pkg, checks its SHA-256 and signer, and installs it.
func (h *Homebrew) Apply(ctx context.Context, _ step.Plan) error {
	pkg := filepath.Join(h.Lock, "Homebrew.pkg")
	h.Out.Say("%s: installing Homebrew %s", stage.Homebrew, brewVersion)
	if _, err := h.M.Run(ctx, "curl", "-fsSL", "-o", pkg, brewURL); err != nil {
		return fmt.Errorf("%s: cannot download %s: %w", stage.Homebrew, brewURL, err)
	}
	defer func() { _, _ = h.M.Run(ctx, "rm", "-f", pkg) }()
	data, err := h.M.ReadFile(pkg)
	if err != nil {
		return fmt.Errorf("%s: cannot read %s: %w", stage.Homebrew, pkg, err)
	}
	want := cmp.Or(h.Sum, brewSHA256)
	if sum := sha256.Sum256(data); hex.EncodeToString(sum[:]) != want {
		return fmt.Errorf("%s: %s's SHA-256 is %x, not the pinned one", stage.Homebrew, pkg, sum)
	}
	out, err := h.M.Run(ctx, "pkgutil", "--check-signature", pkg)
	if err != nil {
		return fmt.Errorf("%s: pkgutil cannot check %s: %w", stage.Homebrew, pkg, err)
	}
	if signer := pkgSigner(string(out)); signer != brewSigner {
		return fmt.Errorf("%s: %s is signed by '%s', not '%s'", stage.Homebrew, pkg, signer, brewSigner)
	}
	if err := h.M.Pass(ctx, true, "installer", "-pkg", pkg, "-target", "/"); err != nil {
		return fmt.Errorf("%s: sudo installer failed on %s: %w", stage.Homebrew, pkg, err)
	}
	if !h.done(ctx) {
		return fmt.Errorf("%s: no brew runs after the install", stage.Homebrew)
	}
	return nil
}

var (
	pkgStatus = regexp.MustCompile(`^\s*Status: .* issued by Apple for distribution$`)
	pkgFirst  = regexp.MustCompile(`^\s*1\. (.*)$`)
)

// pkgSigner returns the signer in the output of pkgutil --check-signature: the first of the chain
// that it prints, after a status of Apple's Developer ID. It returns "" where there is none.
func pkgSigner(out string) string {
	seen := false
	for _, line := range strings.Split(out, "\n") {
		if pkgStatus.MatchString(line) {
			seen = true
		} else if m := pkgFirst.FindStringSubmatch(line); seen && m != nil {
			return m[1]
		}
	}
	return ""
}
