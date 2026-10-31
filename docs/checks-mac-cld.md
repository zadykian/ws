# cld on the Mac

A second workflow, `mac-cld.yml`, sets up cld on another macOS runner, as
[cld and Ghostty](mac-ghostty.md) says, with stand-ins for ws's key and Gateway's app. `--check`
must change nothing. It must note cld's install, and with an empty directory at `WS_GHOSTTY_APP` as
Ghostty's app, Ghostty's files and the ssh lines that may follow too. Without the app, a run must
install cld, whose `cld version` must run, but write no Ghostty settings and give ssh no
`LocalCommand`. With it, a run must write `cld.ghostty` and its line, and `ssh -G ws` give the
watcher's two lines exactly where that cld has `restore tabs`. Stubs then stand in for cld. One
gains `restore tabs` as it updates, so one run must add the lines, which must start its watcher in a
Ghostty pane alone, by zsh and sh. Under one without it, a run must drop them. Each second run and
`--check` must print nothing ([mac-cld-checks.sh](../.github/mac-cld-checks.sh)).
