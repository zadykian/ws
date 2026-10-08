# shell

The `shell` role writes `~/.bashrc` with ble.sh and bash-completion, `PATH` in `~/.profile`, and
`~/.gitconfig`. Each run replaces any change made to them by hand. A machine's own settings go in
files they read where they exist:

- `~/.secrets/env`, which `~/.bashrc` sources first, above its interactive guard, so that
  non-interactive shells get it too;
- `~/.bash_aliases`, which `~/.bashrc` sources too;
- `~/.config/git/local`, which `~/.gitconfig` includes last: a signing key, say.

An installer that appends to `~/.bashrc` or `~/.profile`, a line that adds its directory to
`PATH`, say, loses that line on the next run: move it to `~/.bash_aliases`.

`~/.gitconfig` has gh's credential helper already, as `gh auth setup-git` writes it, so running
that changes nothing. A dry run shows no diff of `~/.gitconfig`, as one set up by hand may hold a
credential. It names instead the keys a real run would drop, without their values: move them to
`~/.config/git/local` first. A subsection shows as `*`, as in `http.*.extraheader`, since one may
be a URL, which names a host. A real run keeps the old file beside the new one, as
`~/.gitconfig.*~`.
