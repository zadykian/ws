# The Mac's checks on a clean macOS 27

GitHub's runner is macOS 26.7.2. It drops the `apps` role and the wallpaper's unit, has no display,
and never logs out. So what no doc settles is tried on the Mac itself, reset to a clean macOS 27, in
one sitting, once every sub-issue of #1533 is on `main`. These are the ten trials of #1578. None has
run: the build had no Mac. A failure changes the role that its trial names, never the trial.
The scripts run nothing in CI, but [checks-test.sh](checks-test.sh) tries their functions in `lint`,
against stand-ins for the tools and on the real roles' files.

## The scripts

Each prints a line a check: `ok`, `FAIL` with what it saw, `info` with a value to read, or `wait`
for a check that needs a value or a step by hand first. It exits 1 where any fails. A script only
reads. The first run, the second and each step by hand are the person's, as the page says.

| Script | Runs | Trials |
| --- | --- | --- |
| [setup.sh](setup.sh) `[CHECK...]` | As the user. `sudo` reads the login items. | 1, 5, 10. |
| [prefs.sh](prefs.sh) `[CHECK...]` | As the user. | 2, 3, 4, 6, 7. |
| [tools.sh](tools.sh) `8 [FILE]`, `9 [SHA256]` | As the user. | 8, 9. |
| [lib.sh](lib.sh) | Sourced by the three. | Setup's plan and status, the roles' apps. |

`WS_TREE` names ws's tree, `/opt/ws` by default. The scripts read the Dock's apps, the login items,
the formulae and the hotkeys from its roles' files, so a change of a role reaches them.

## Before the trials

1. Reset the Mac to a clean macOS 27. Sign in with an Apple account and let macOS finish its first
   boot. Make no change of the settings that the trials read.
2. Run the runbook's steps up to the bootstrap, as [the runbook](../runbook.md) says, with the
   values of the Mac's servers. Give sudo the user's password where it asks.
3. Take a Mac with a display, a second display and a YubiKey for trials 1 and 7.
4. Note the time, the Mac's model and `sw_vers`, to put beside a failure.

## What the runner proved

The runner is macOS 26.7.2 on arm64. Its probes P0 to P2 ran the first run twice, with `--plan` at 0
after the first and no sudo in the second (cases 28 and 96). They read back desktop's preferences by
type (82), devtools (83), docker's config (84), the zsh clones and the hook (87) and power's units
(88). Case 81, the hotkeys, landed after the last probe: the runner has not read it back. The runner
holds no login item and no display, so the wallpaper and the apps are proved on the Mac alone. A
trial below names what the runner leaves open.

## The trials

Run them in this order. Trials 1 and 2 first, as a first run puts every later trial's state in place.

**1. The first run installs everything, and the Mac then converges.**

- Where: the Mac. Run `ws macos setup` as the user, on a terminal, and answer no prompt: it passes
  `--yes`. After the first run's end, reboot as logi-options+ asks, log in, log out and in once
  more, and run `setup.sh 1`. Then `sudo -k` and run `ws macos setup` a second time.
- It checks: setup's plan exits 0 in both envs. mise's status finds nothing missing. Each formula
  stands in `/opt/homebrew/opt`, each app of the Dock and the login items is installed, and the
  Dock holds its nine apps in order. A login shell's `go` is mise's Go 1.27.3, asked from the home
  directory. `ykman` runs, and docker's command line and plugins answer.
- Passes: every line is `ok`, and the second run asks for no password and ends with the plan at 0.
- By hand: a shell that was open before the first run finds no `go` or `rg` until its user logs in
  again. `ykman list` shows the YubiKey. logi-options+ asks for sudo and a reboot, which is a
  prompt of its installer. The apps are `Telegram Desktop.app` and `Yandex Telemost.app`, as the
  casks name them. `/Applications/Safari.app` exists.
- Open on the runner: macOS 27's bottles. mise's tags end at macOS 26, so a formula takes Tahoe's
  bottle, and Homebrew's API lists `arm64_golden_gate` too. The bottles of `ykman`'s `python@3.15` and
  of `wallpaper` must pour.
- A failure changes: the role named by status, or the formula's pin in its `mise.macos.toml`. If
  `-E ws,apps` does not expand the set `ws` as the server's `-E ws,secrets` does, setup's two runs
  ([setup.go](../cli/setup.go)). If `/usr/local/go` fails as the user, devtools.

**2. Each preference has the type that its role writes, and a second status finds them in sync.**

- Where: the Mac, after the log out of trial 1. `prefs.sh 2`.
- It checks: `defaults read-type` and the value of 30 preferences of desktop and keyboard, the
  screen saver's delay of the current host, and mise's status.
- Passes: every line is `ok`. The four in doubt print an `info` line each: the beep's
  `com.apple.sound.uiaudio.enabled` in `NSGlobalDomain`, `target-screenshot`, `target-screenrecording`
  and `NSGlassTintAmount`.
- By hand: a screenshot goes to the clipboard and a recording to a file. The glass tint shows after
  a log in. `NewWindowTarget` `PfHm` opens Finder at the home folder with no `NewWindowTargetPath`.
  Appearance stays Dark by day, with its Auto key false. The Dock's size is 74 and Finder lists in
  columns. The beep is silent and the interface sounds are off.
- A failure changes: [desktop](../roles/desktop/README.md)'s `mise.macos.toml`: a key's value
  type, or its domain, where macOS reads another. A key that the Dock or Finder rewrites on exit goes
  out of the role with a reason on its page.

**3. The layout types at the login window and after it.**

- Where: the Mac, after the log out of trial 1. `prefs.sh 3`.
- It checks: the files of `/Library/Keyboard Layouts/ws.bundle` are root's at 644
  and equal the role's, and the bundle holds no other. `com.apple.inputsources` lists `ENG`,
  -20611, and `RUS`, -21268, as integers and strings. `com.apple.HIToolbox` lists four sources, the
  layout's two by their IDs first.
  The input source in use is one of the layout's two. Setup's plan exits 0.
- Passes: every line is `ok`. By hand, type at the login window, then after the log in. Switch
  the input source with Command+Space, and run `prefs.sh 3` again. The plan must still exit 0.
- Open: whether macOS keeps both arrays as mise wrote them after a log in and a switch, or adds
  a layout of Apple's back to `com.apple.HIToolbox`. The role takes `InputSourceKind` of
  `com.apple.PressAndHold` to be `Non Keyboard Input Method`: read what macOS writes.
  `prefs.sh 3` prints the input source in use.
- A failure changes: [keyboard](../roles/keyboard/README.md)'s input sources. If macOS rewrites an
  array of `com.apple.HIToolbox` or `com.apple.inputsources`, the role drops that key from its
  entries, and the runbook names the sources by hand.

**4. Each hotkey holds.**

- Where: the Mac, after a log in. `prefs.sh 4`.
- It checks: IDs 61, 64 and 160 are on at their keys, `enabled` is a boolean, Caps Lock maps to
  Control, and each of the 63 off IDs reads `enabled` false.
  It prints the off IDs that macOS lacked, which mise made.
- Passes: every line is `ok`. By hand, Command+Space switches the input source, Option+Space opens
  Spotlight, and Caps Lock acts as Control.
- By hand, note: what Command+L opens, and whether Zed and Safari lose their own Command+L.
  Press the keys of one hotkey that the role turns off, and see that nothing answers.
- Open: that a log in applies the hotkeys and Caps Lock with no `activateSettings`, and that 34 and
  98 stay as macOS has them. A dictionary with a key more than `enabled` and `value` in 61, 64 or
  160 would leave the status different for good: `prefs.sh 2` shows it.
- A failure changes: keyboard's entries. A hotkey that macOS keeps on goes out of the off list.

**5. The login items open once, and the agents run.**

- Where: the Mac, after a log in. `setup.sh 5`.
- It checks: `sudo sfltool dumpbtm` holds one login item for each of the six apps. mise's status
  holds them. Secretive's agent, Logi's and Rectangle Pro run.
- Passes: every line is `ok`. By hand, System Settings, General, Login Items shows the six items,
  and an item that Rectangle Pro or Claude added at its first launch is gone: `exclusive` removed it.
- If Secretive's agent is a login item that `sfltool` shows and mise's status does not, add its app
  to `apps` in `mise.apps.toml`. mise reads `LSSharedFileList` alone. The records' format is
  macOS's, so where every count is 0, read the dump first: the match is the script's.
- A failure changes: [apps](../roles/apps/README.md)'s `login_items`.

**6. Updates are off, and the display stays on.**

- Where: the Mac. `prefs.sh 6`.
- It checks: `pmset -g custom` shows `displaysleep 0` under AC Power, and the App Store's
  `AutoUpdate` is 0. It prints the keys of `com.apple.SoftwareUpdate` and `com.apple.commerce`.
- Passes: both lines are `ok`. By hand, System Settings, General, Software Update, Automatic
  Updates shows "Install application updates" off.
- A failure changes: [power](../roles/power/README.md). Apple marks `AutomaticallyInstallAppUpdates`
  removed on macOS 27. Where another key rules, change power's check and command, the page's "macOS
  27" part and case 88 of CI. `pmset -c displaysleep 0` and the check's awk read macOS 27's text.

**7. The wallpaper is Stone on a second Space and on a second screen.**

- Where: the Mac. `prefs.sh 7`, on the first Space, on a second Space, and with a second screen.
- It checks: `wallpaper get` prints a line a screen, each Stone.
- Passes: every line is `ok` on each Space and with both screens. `wallpaper set` shows Stone by
  eye, after the first run.
- Open: the unit sets Stone on the current Space alone. A second Space and a screen that joins
  later may keep macOS's picture. Then the unit sets it again for each, or the page says so.
- A failure changes: the `state/wallpaper` unit of [desktop](../roles/desktop/README.md).

**8. Containers run under the firewall, and a torrent downloads.**

- Where: the Mac. Run `colima start`, then `docker run --rm hello-world`, then
  `docker compose version` and `docker buildx version`. Download a torrent in Transmission.
  `tools.sh 8 PATH_OF_THE_FILE` reads the state.
- It checks: the firewall blocks all incoming, colima runs, docker reaches its engine, the image
  that the run pulled is there, and the torrent's file holds data.
- Passes: every line is `ok`. The run pulls through the Keychain, by `osxkeychain`, and the
  plugins are found by `config.json`. Transmission asks for no firewall exception: replies pass block-all.
- A failure changes: [docker](../roles/docker/README.md)'s `config.json` and formulae, or
  [firewall](../roles/firewall/README.md)'s page where block-all stops a reply.

**9. claude and gpg.**

- Where: the Mac. `tools.sh 9`, which prints claude's binary and its sum. Run `claude update`, then
  `tools.sh 9 SUM`, SUM being the sum from before. For gpg, with a key, `echo x | gpg --clearsign`.
- It checks: `claude` runs, resolves into a Caskroom, and keeps its sum. `gpg-agent.conf` names
  `pinentry-mac`, which is executable, and gpg's agent answers.
- Passes: every line is `ok`. By hand, gpg asks for its passphrase in a window of `pinentry-mac`,
  and `claude` links into `/opt/homebrew/bin`.
- A failure changes: [apps](../roles/apps/README.md)'s `claude-code` cask, or devtools'
  `gpg-agent.conf`. If claude's updater replaces the binary, the role says how to turn it off.

**10. A new shell loads the three plugins, and oh-my-zsh asks no update.**

- Where: the Mac. `setup.sh 10`, in a terminal that opened after the first run.
- It checks: `plugins` is git, zsh-autosuggestions and zsh-syntax-highlighting, both plugins' own
  functions exist, oh-my-zsh's update is `disabled`, and the shell prints no compaudit warning.
- Passes: every line is `ok`. By hand, in Ghostty, drop an ssh and see that the tab resets the
  terminal's modes. The hook runs first of `precmd`.
- A failure changes: [zsh](../roles/zsh/README.md)'s `zshrc` or its clones' pins.
