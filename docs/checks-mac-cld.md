# Ghostty and cld on the Mac

A second workflow, `mac-cld.yml`, sets up Ghostty, Zed and cld on another macOS runner, as [Ghostty
and cld on the Mac](mac-ghostty.md) and [Zed on the Mac](mac-zed.md) say. It has stand-ins for ws's
key, and a file key that `mac.conf` names for ws's login. On a runner without Ghostty, Zed, its font
or cld, `--check` must change nothing, and note the casks, cld, the ssh lines and the files of
Ghostty and Zed. Where the casks `ghostty` and `zed` fail, a run must warn and install cld alone. A
run must then install the casks, and write `ws.ghostty` as `mac/ws.ghostty` under the header, and
its line, once. Ghostty's `+validate-config` must take the config, and `+list-fonts` find its font.
`ssh -G ws` must give the watcher's lines exactly where cld has `restore tabs`. A stub cld that
gains it as it updates must have a run add them, whose line starts its watcher in a Ghostty pane
alone. Under one without it, a run must drop them. Each second run and `--check` must print nothing
([mac-cld-checks.sh](../.github/mac-cld-checks.sh)).
