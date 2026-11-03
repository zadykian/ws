# Another repository on ws

`WS_HOOK` names a release of another repository on GitHub, as `OWNER/REPO@vX.Y.Z`, whose command
sets up more of ws, such as what this public repository leaves out. The tag is required. The
repository needs releases, each with `REPO-linux-ARCH` for `amd64` and `arm64`, and `REPO.sha256`,
which lists the SHA-256 of each build. `ws mac apply` takes the key, and reads it not. The command
needs no secret from the Mac: ws's secrets file and gh's login are ws's own
([The secrets file](shell.md#the-secrets-file), [gh](devtools.md#gh)).

The stage downloads the build of ws's architecture and the sums by `gh release download`, with the
Mac's gh login. ws has no clone and no gh login for it. The Mac checks the build's SHA-256 against
the sums, and a mismatch or a missing release stops the stage, naming the tag. The files script,
`mac/wizard-remote-files.sh`, takes the build over `ssh ws-ssh` into `/var/lib/ws-wizard/bin/REPO-TAG`,
mode 700 in a directory of mode 700, root's alone, and checks the SHA-256 again. A new tag replaces
the old one's file. A new tag is a new value, so `--update` sends it.

Then `REPO apply` runs as root over `ssh ws-ssh`, never `ssh -t`, so it neither prompts nor echoes
its stdin, which is empty. The Mac's Keychain is not read. That run leaves a stamp,
`/var/lib/ws-wizard/hook-REPO` on ws, which names the tag, and the stage is done. A second run
leaves the command alone, but `--update` runs it again. A run that fails stops the wizard, and the
next runs it again. `--check` runs `REPO apply --check` where ws holds the tag's build, and
otherwise names what the stage would do.

The stage Hook on the Mac comes first. It clones the repository in `~/repository/REPO` by gh's login
on the Mac, where missing, and never pulls it. Another repository's clone there stops the stage.
Where the clone has `.githooks`, git's `core.hooksPath` names it by an absolute path, as in
a clone of ws ([The hooks](checks.md#the-hooks)). Where it has a `Brewfile` whose entries Homebrew
lacks, as `brew bundle check`, run with Homebrew's auto-update off, finds, `brew bundle` installs
them, and upgrades nothing. `--check` names each command that the stage would run.

The stage Word list copies ws's word list, which the hooks read ([The hooks](checks.md#the-hooks)),
with `WS_HOOK` or without, after Hook on ws, whose command may write it. Where ws has one, and
the Mac's differs by git's hash or can't be read, the files script, `mac/wizard-remote-files.sh`,
sends it from ws. It replaces the Mac's, mode 600, but a link there stops the stage, `--check` too,
naming its target. The wizard never prints the list: `--check` says it would copy it, with no diff.
