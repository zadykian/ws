# Ghostty and cld on the Mac

A second workflow, `mac-cld.yml`, sets up Ghostty and cld on another macOS runner, as
[Ghostty and cld on the Mac](mac-ghostty.md) says. It has stand-ins for ws's key and Gateway's app.
On a runner without Ghostty, its font or cld, `--check` must change nothing, and note both casks,
cld, the ssh lines and Ghostty's files. Where the cask `ghostty` fails, a run must warn and install
cld alone. A run must then install both casks, and write `ws.ghostty` as `mac/ws.ghostty` under the
header, and its line, once. Ghostty's `+validate-config` must take the config, and `+list-fonts`
find its font. `ssh -G ws` must give the watcher's lines exactly where cld has `restore tabs`. A
stub cld that gains it as it updates must have a run add them, whose line starts its watcher in a
Ghostty pane alone. Under one without it, a run must drop them. Each second run and `--check` must
print nothing ([mac-cld-checks.sh](../.github/mac-cld-checks.sh)).
