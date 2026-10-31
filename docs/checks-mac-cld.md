# cld on the Mac

A second workflow, `mac-cld.yml`, sets up Ghostty and cld on another macOS runner, as
[cld and Ghostty](mac-ghostty.md) says, with stand-ins for ws's key and Gateway's app. On a runner
without Ghostty or cld, `--check` must change nothing, and note the cask, cld, the ssh lines and
Ghostty's files. Where the cask `ghostty` fails, a run must warn and install cld alone. A run must
then install the cask, and write `cld.ghostty` and its line, and `ssh -G ws` give the watcher's two
lines exactly where that cld has `restore tabs`. Stubs then stand in for cld. One gains `restore
tabs` as it updates, so one run must add the lines, which must start its watcher in a Ghostty pane
alone, by zsh and sh. Under one without it, a run must drop them. Each second run and `--check`
must print nothing ([mac-cld-checks.sh](../.github/mac-cld-checks.sh)).
