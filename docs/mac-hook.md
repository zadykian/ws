# Another repository on ws

`WS_HOOK` names another repository on GitHub, as `OWNER/REPO` or `OWNER/REPO@vX.Y.Z`, whose `apply.sh`
sets up more of ws, such as what this public repository leaves out. `WS_HOOK_SECRET` names the
Keychain item of a secret that the script reads. `ws mac apply` takes both keys, and reads neither.
Where ws lacks `/root/repository/REPO`, the stage sends it the Mac's `origin/main` as a git bundle,
which the files script clones on branch `main` with no gh, `origin` at its GitHub URL. A `REPO` of
`ws`, or another repository's clone there, stops the stage. Without a tag it never pulls a clone,
as the script updates its own, and a clone detached at a tag stays so. With one, the stage fetches
the tag on the Mac, never moving one that the Mac has, so that a tag moved on GitHub stops it,
naming the tag. It sends that tag alone, and the files script detaches ws's clone at it, but refuses
a clone with a change to a tracked file. A new tag is a new value, so `--update` sends it.

Where the Keychain lacks the item, it runs `security add-generic-password`, which asks for its
secret twice at the terminal. Where `security find-generic-password -w` can't read it, the stage
stops, naming its error. Then `apply.sh` runs as root over `ssh ws-ssh`, never `ssh -t`, so it
neither prompts nor echoes its stdin: what security prints, then the end of input, or empty without
an item. The wizard never reads the secret, and passes no arguments. That run leaves a stamp,
`/var/lib/ws-wizard/hook-REPO` on ws, and the stage is done. A second run leaves `apply.sh` alone,
but `--update` runs it again. A run that fails stops the wizard, and the next runs it again. An item
whose access list lacks `security` asks at both of the stage's reads, which "Always Allow" answers.

The stage Hook on the Mac comes first. It clones the repository in `~/repository/REPO` by gh's login
on the Mac, where missing, and never pulls it. Another repository's clone there stops the stage, as
on ws. Where the clone has `.githooks`, git's `core.hooksPath` names it by an absolute path, as in
a clone of ws ([The hooks](checks.md#the-hooks)). Where it has a `Brewfile` whose entries Homebrew
lacks, as `brew bundle check`, run with Homebrew's auto-update off, finds, `brew bundle` installs
them, and upgrades nothing. `--check` names each command that the stage would run.

The stage Word list copies ws's word list, which the hooks read ([The hooks](checks.md#the-hooks)),
with `WS_HOOK` or without, after Hook on ws, whose `apply.sh` may write it. Where ws has one, and
the Mac's differs by git's hash or can't be read, the files script, `mac/wizard-remote-files.sh`,
sends it from ws. It replaces the Mac's, mode 600, but a link there stops the stage, `--check` too,
naming its target. The wizard never prints the list: `--check` says it would copy it, with no diff.
