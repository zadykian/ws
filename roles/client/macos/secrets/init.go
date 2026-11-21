package secrets

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"
	"time"

	"filippo.io/age"

	"github.com/zadykian/ws/engine/core/effect"
)

// fnoxBin is fnox where state.fnox copies it. ws runs it by PATH, with /usr/local/bin first.
const fnoxBin = "/usr/local/bin/fnox"

// The topic and the titles of init's pull requests: the first run's, and a later one's that
// declares new inputs.
const (
	initTopic   = "init"
	startTitle  = "chore: set up the recipients and the inputs"
	inputsTitle = "chore: declare the inputs of a newer release"
)

// Init sets the secrets repository up, as ws client secrets init does. It writes the Nano's
// and a backup's recipients, and declares each input of the tree's server roles, empty.
type Init struct {
	M      effect.Machine
	Getenv func(string) string
	// Out takes what init says, the backup's secret line among it. In answers once that line is kept.
	Out io.Writer
	In  io.Reader
	// Tree is ws's tree, whose mise.toml names the repository and whose roles declare the inputs.
	Tree string
	// NewBackup, Sleep and Now are the backup's maker and the clock of the checks' wait: nil takes
	// the real ones.
	NewBackup func() (*age.X25519Identity, error)
	Sleep     func(context.Context, time.Duration) error
	Now       func() time.Time
}

// Run sets the repository up in a temporary clone, and lands the change by a pull request. A run
// that finds both recipients and every input lands nothing. A pull request of init's that is still
// open stops it: a second one would make another backup, and both would add fnox.toml.
func (i Init) Run(ctx context.Context) error {
	repo, err := RepoOf(i.M, i.Tree)
	if err != nil {
		return err
	}
	inputs, err := Inputs(i.M, path.Join(i.Tree, "roles/server/ubuntu"))
	if err != nil {
		return err
	}
	for _, tool := range []string{fnoxBin, Plugin} {
		if _, err := i.M.Stat(tool); err != nil {
			return fmt.Errorf("no %s: run ws client setup first, which puts fnox and "+
				"age-plugin-yubikey in /usr/local/bin", tool)
		}
	}
	keyFile, err := KeyFile(i.Getenv)
	if err != nil {
		return err
	}
	c, err := Open(ctx, i.M, repo)
	if err != nil {
		return err
	}
	defer c.Close(ctx)
	c.FnoxEnv, c.Sleep, c.Now = FnoxEnv(i.Getenv, keyFile), i.Sleep, i.Now
	open, err := c.OpenPulls(ctx, initTopic)
	if err != nil {
		return err
	}
	if len(open) > 0 {
		return fmt.Errorf("init's pull request %s is still open in %s: merge it once its checks pass, or close it "+
			"and run init again, which makes a new backup", strings.Join(open, " and "), repo.Slug())
	}
	conf, found, err := ReadConfig(i.M, path.Join(c.Dir, SharedFile), SharedProvider)
	if err != nil {
		return err
	}
	if lack := lackedLabels(conf); found && len(lack) > 0 {
		return fmt.Errorf("%s of %s has no recipient %s in [providers.%s.recipients]: init adds none to a "+
			"started file, as a new recipient needs every value encrypted again", SharedFile, repo.Slug(),
			strings.Join(lack, " or "), SharedProvider)
	}
	slot, err := NanoIdentity(ctx, i.M, i.Out, conf.Recipients[NanoLabel])
	if err != nil {
		return err
	}
	if err := i.keepStub(ctx, slot, keyFile); err != nil {
		return err
	}
	var backup *age.X25519Identity
	if !found {
		if backup, err = i.start(c, slot); err != nil {
			return err
		}
	}
	added, err := declare(ctx, c, inputs, conf.Secrets)
	if err != nil {
		return err
	}
	if found && len(added) == 0 {
		_, _ = fmt.Fprintf(i.Out, "%s holds both recipients and every input: nothing to land\n", repo.Slug())
		return nil
	}
	if backup != nil {
		if err := i.showBackup(ctx, backup); err != nil {
			return err
		}
	}
	ch := Change{Topic: initTopic, Title: inputsTitle, Files: []string{SharedFile},
		Body: "The inputs that a newer release of ws declares, each empty: " + strings.Join(added, ", ") + "."}
	if !found {
		ch.Title, ch.Body = startTitle, "The recipients nano and backup, and the inputs, each empty."
	}
	url, err := c.Land(ctx, i.Out, ch)
	if err == nil && url == "" {
		// fnox wrote the inputs elsewhere, so the repository would lack them.
		return fmt.Errorf("fnox set left %s of %s as it was, so nothing landed: %s stay undeclared", SharedFile,
			repo.Slug(), strings.Join(added, ", "))
	}
	return err
}

// lackedLabels are the labels of the Nano's and the backup's recipients that conf lacks.
func lackedLabels(conf Config) []string {
	var lack []string
	for _, label := range []string{NanoLabel, BackupLabel} {
		if _, ok := conf.Recipients[label]; !ok {
			lack = append(lack, label)
		}
	}
	return lack
}

// keepStub keeps the stub of the Nano's slot in fnox's key file, so that fnox finds the Nano.
func (i Init) keepStub(ctx context.Context, slot Slot, keyFile string) error {
	stub, err := Stub(ctx, i.M, slot.Slot)
	if err != nil {
		return err
	}
	wrote, err := KeepStub(i.M, keyFile, stub)
	if wrote {
		_, _ = fmt.Fprintf(i.Out, "kept the stub of the Nano's slot %d in %s\n", slot.Slot, keyFile)
	}
	return err
}

// start makes the backup key and writes a new fnox.toml of its recipient and the Nano's. The
// backup's secret stays in memory until showBackup prints it.
func (i Init) start(c *Clone, slot Slot) (*age.X25519Identity, error) {
	newBackup := i.NewBackup
	if newBackup == nil {
		newBackup = age.GenerateX25519Identity
	}
	backup, err := newBackup()
	if err != nil {
		return nil, fmt.Errorf("cannot make the backup key: %w", err)
	}
	text, err := Start(slot.Recipient, backup.Recipient().String())
	if err != nil {
		return nil, err
	}
	file := path.Join(c.Dir, SharedFile)
	if err := i.M.WriteFile(file, text, 0o644); err != nil {
		return nil, fmt.Errorf("cannot write %s: %w", file, err)
	}
	return backup, nil
}

// declare declares each input that the secrets lack, in fnox.toml, as an empty value. A server's run
// then finds each one that mise asks for, and fails on a key that opens none. It returns the names
// it declared.
func declare(ctx context.Context, c *Clone, inputs, secrets []string) ([]string, error) {
	var added []string
	for _, name := range inputs {
		if slices.Contains(secrets, name) {
			continue
		}
		// fnox set reads the value from stdin, and stores none as an empty one.
		if _, err := c.Fnox(ctx, []byte{}, "set", name, "--provider", SharedProvider); err != nil {
			return nil, fmt.Errorf("cannot declare the input %s in %s: %w", name, SharedFile, err)
		}
		added = append(added, name)
	}
	return added, nil
}

// showBackup prints the backup's secret line once, and waits for the user to keep it. It comes
// before the push: a merged fnox.toml whose backup was never shown would have none. ctx's end stops
// the wait, as a Ctrl-C reaches ws as ctx's end alone.
func (i Init) showBackup(ctx context.Context, backup *age.X25519Identity) error {
	_, _ = fmt.Fprintf(i.Out, "\nThe backup key opens every value where the Nano is lost. ws prints its secret line "+
		"this once and writes it nowhere: keep it on paper or in a password manager.\n\n%s\n\n"+
		"Press Enter once it is kept.\n", backup)
	// The read stays blocked past ctx's end, until the process ends.
	answered := make(chan error, 1)
	go func() {
		_, err := bufio.NewReader(i.In).ReadString('\n')
		answered <- err
	}()
	var err error
	select {
	case err = <-answered:
	case <-ctx.Done():
		err = ctx.Err()
	}
	if err != nil {
		return errors.New("init stopped before the push, with no answer that the backup's line is kept: " +
			"nothing landed, and the next run makes a new backup")
	}
	return nil
}
