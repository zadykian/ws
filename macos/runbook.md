# The Mac's runbook

A person does these steps in this order, as no wizard asks them. Each step is done once the check
beside it holds. Run `ws macos setup` where a step says so: it sets up the Mac's end ([The
Mac](README.md#setup)). The tree's `mise.toml` sets `locked = true`, so mise takes each tool as the
tree's lock holds it and refuses one that the lock lacks.

## Tools

Run `install.sh` through sudo, by the three lines of
[Install](../.github/workflows/release.md#install), `set -o pipefail` among them. It downloads
`ws-darwin-arm64`, checks it against the release's `ws.sha256`, and installs it as
`/usr/local/bin/ws`, root's and 755. It makes `/usr/local/bin` where a new Mac lacks it, and
refuses a `/usr/local` or `/usr/local/bin` that another user owns or may write: it names the chown.
`ws macos setup` takes the rest: the command line tools, mise by its SHA-256, the tree in
`/opt/ws`, the miserc and the plugin's link ([The bootstrap](#the-bootstrap)).

## Answers

Make the Mac's `/etc/ws/values.yml`, as root, through sudo: empty on a new Mac, as setup writes
its keys. Write each server's values in `/etc/ws/servers/NAME/values.yml`, as root too ([The two
files](values/README.md#the-two-files)). The files' check refuses a clash.

1. Pick a slot, 0 to 15, that no server has: `WS_SLOT`.
2. Pick the ports: 51820 for the tunnel and 22022 for the router's forward to ws's 22. Where
   `WS_REMOTE` repeats on another server, add the slot to each. A guest's `WS_REMOTE` is its host's:
   it takes a `WS_TUNNEL_PORT` of its own and no `WS_REMOTE_PORT`.
3. Set the rest that differs from its default: `WS_TARGET`, `WS_LAN`, `WS_LAN_ADDRESS`,
   `WS_REMOTE`, `WS_REMOTE_PORT`, `WS_TUNNEL_NETWORKS` and `WS_RESOLVERS`. The
   [router's page](../ubuntu/roles/cable/router.md) gives the LAN's defaults.
4. For a guest of a host, write the host's file first, the host being a server too, at slot 0.
   The guest's file holds `WS_HOST` with the host's name, its slot from 1, its `WS_TUNNEL_PORT` and
   what it routes. It holds none of the keys above that name a way
   ([A guest](values/README.md#a-guest)).

## The bootstrap

Run `ws macos setup` as the user, after the values above, and not as root: it asks
sudo for what root's paths need. It refuses a bad value before any write, naming the key and its
file. It then does what a person did by hand, in this order:

1. Where `xcode-select -p` finds no command line tools, it runs `xcode-select --install`, which
   opens Apple's dialog, and waits until the tools stand. Ctrl-C stops the wait.
2. It installs mise 2026.11.9 in `/usr/local/bin`, checked against its SHA-256, and writes
   `/etc/mise/miserc.toml` and `/etc/mise/config.toml`, which trusts the tree.
3. It checks this ws's release tag out in `/opt/ws`, root's, which no one else may write, and links
   the tree's plugin into `/usr/local/share/mise/plugins`.
4. It runs mise's bootstrap in `/opt/ws/macos` with `--yes`, so that it asks nothing before it
   writes the preferences. The bootstrap installs gh and sets up ssh, git, cld and the Mac's other
   roles ([The Mac's frame](roles/frame/README.md)). It makes the Mac's WireGuard key, whose public
   half is `/etc/ws/tunnel.pub`.
5. It runs the bootstrap again with `-E ws,apps`, which installs the apps and sets the login items.
   A failure there leaves ws's units set up, and fails setup, as mise ends a run at its first
   failed step ([The apps' env](roles/frame/README.md#the-apps-env)).

A second run changes nothing and asks no password. `ws macos setup --plan` plans these steps, and
mise's bootstrap where none is due, in both envs, by its plan and then its status. The plan covers no
preference, Dock app or login item, and the status does, with the tools, repositories and
packages. It exits 0 where nothing would change and 2 where something would. A ws that names no
release tag is refused: `--in-place` runs the tree in `/opt/ws` as it stands, after the guard. So
is a ws of another version than `/usr/local/bin/ws`, which the units run, or a missing file there:
`ws update --version X.Y.Z` installs that release there. A tree, or a directory above it, that
others may write is refused too ([Setup](README.md#setup)). `ws update`, as the user, brings ws
to the latest release, or to `--version`'s, and runs that ws's setup, with sudo for the write
([Update](../.github/workflows/release.md#update)).

## GitHub

1. Run `gh auth login --scopes write:public_key,write:ssh_signing_key`, at github.com's device flow.
2. Add the key for ws with `gh ssh-key add` and the login key's `.pub`.
3. Add git's signing key with `gh ssh-key add --type signing` and its `.pub`, titled with the
   Mac's name ([The signing key](roles/git/README.md#the-signing-key)).

GitHub is the only source of the server's keys: nothing copies a key to the server. Root takes the
account's keys from GitHub, as [the server's runbook](../ubuntu/runbook.md) says, so run this step
before it. A cloud VM takes the keys its provider holds, which must be the account's.

## The secrets

1. Once, create the secrets repository by `gh repo create zadykian/ws-secrets --private`, with its
   README, license and `check` workflow, whose jobs `values` and `gitleaks` its ruleset on `main`
   requires.
2. With the Nano plugged in, run `ws macos secrets init`. Where the Nano holds no age
   identity, it asks for the Nano's PIN and a touch. It prints the backup key's secret line once:
   keep it on paper or in a password manager, then press Enter. It lands `fnox.toml` by a pull
   request ([ws macos secrets init](roles/secrets/README.md#ws-macos-secrets-init)).
3. Run it again once `/opt/ws` moves to a release that declares a new input. Until then, a
   server's run of that release fails at mise, which finds no value of the input.
4. To set a value, such as `WS_GITHUB_TOKEN`, clone the repository by
   `gh repo clone zadykian/ws-secrets`, and switch to a new branch in the clone. Run
   `fnox set NAME --provider age` for a value that every server shares, or
   `fnox -P HOSTNAME set NAME --provider host` for one server's, once add has made its profile.
   Give the value on standard input, never as an argument. Commit the changed file, signed, push
   the branch and open a pull request by `gh pr create`. Once `values` and `gitleaks` pass, merge
   it by `gh pr merge --merge --delete-branch`. Then, for each server NAME that reads the value,
   run `ws macos secrets add NAME`, which runs the setup with the secrets there, through its host
   for a guest ([ws macos secrets add](roles/secrets/README.md#ws-macos-secrets-add),
   [The task](../ubuntu/roles/secrets/README.md#the-task)).

## ws's host key

A guest has none to read: the host answers for it, and `ws macos secrets add NAME` records the
guest's key. Read the host's, as below.

1. Read ws's key with `ssh-keyscan -t ed25519 ADDRESS`, ADDRESS being ws's LAN address, or the
   router's name and port off the LAN. A new ws has a lease from the router's pool.
2. At ws's console, `ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub` prints ws's own
   fingerprint. Compare it with the scan's, from `ssh-keygen -lf -`.
3. Where they match, append the scan's line to `~/.ssh/known_hosts`, with NAME, the server's name, in
   place of the address. A new ws faces the internet until its `security` role runs.

## Login

Log in with `ssh -o HostName=ADDRESS NAME`, as the tunnel is not up yet. It must log in as root by
the Mac's key, which root took from GitHub ([The server's runbook](../ubuntu/runbook.md)). A cloud
image whose root refuses a login takes `disable_root: false` and `users: []` in its cloud-config.
Where ssh names another host key, drop the old one with `ssh-keygen -R NAME`. A guest has no
login before its tunnel: log in to its host, which runs `incus exec NAME -- bash` for it.

## ws on ws

1. At ws, write the values that differ from their defaults first, as
   [the server's runbook](../ubuntu/runbook.md) says. At slot N other than 0, set
   `WS_TUNNEL_ADDRESS` to `10.99.N.1/24`, `WS_TUNNEL_PORT` to the Mac's, and
   `WS_TUNNEL_PEERS` to `mac` at `10.99.N.2`
   ([The tunnel](../ubuntu/roles/tunnel/README.md#the-tunnel)). A guest's go in its own values on
   its host, `/etc/ws/guests/NAME/values.yml`.
2. Run `install.sh` by the three lines of [Install](../.github/workflows/release.md#install),
   `set -o pipefail` among them, and then `ws ubuntu guest setup`. Check ws with `ws version`.
3. For a guest, run the host's setup first and the guest's through it
   ([The guests of a host](../ubuntu/runbook.md#the-guests-of-a-host)). Run
   `ws macos secrets add NAME --address ADDRESS`, with `--port PORT` off the LAN, ADDRESS being
   the host's for a guest. It adds ws's host key to the secrets and as the repository's deploy
   key, with the Nano's PIN and a touch. It then runs the server's bootstrap with the secrets
   over `ssh -t`. A first run takes 20 to 40 minutes, and a dropped ssh ends it: run the command
   again. `ws macos secrets init` runs once before, as [The secrets](#the-secrets) says
   ([ws macos secrets add](roles/secrets/README.md#ws-macos-secrets-add)).

## claude

Run `ssh -t NAME claude auth login`, and open the address it shows in the Mac's browser.

## Signing key

Add the server's signing key by hand in `@zadykiand`'s settings on GitHub, as a signing key. Its
`.pub` is the file that `~/.gitconfig` there names. The Mac's gh is the maintainer's login and adds
none for the machine account ([Signing](../ubuntu/roles/git/README.md#signing)).
Where the same key stands in `@zadykian`'s settings as a signing key, take it off there. GitHub
would show a commit that ws signs under the maintainer's email as verified.

## The tunnel

1. Copy the Mac's `/etc/ws/tunnel.pub` to ws's `/etc/wireguard/mac.pub`, as root, and run the
   server's bootstrap there, as [its runbook](../ubuntu/runbook.md) says
   ([The tunnel](../ubuntu/roles/tunnel/README.md#the-tunnel)). For a guest, copy it to its host's
   `/etc/wireguard/wgN/mac.pub`, N the guest's slot, and run the host's setup
   ([The Mac's tunnel](README.md#the-macs-tunnel)).
2. Put wg0's public key, which that page names, in `WS_PUBLIC_KEY` of the server's `values.yml`.
3. Run `ws macos setup` again, and allow ws-tunnel in the Background.
4. Ping the server's tunnel address, `10.99.N.1` for slot N, and run `ssh NAME`. A handshake must
   come within 30 s.

## The cable

Skip it on a cloud VM. A guest follows its host's: the host's file alone names the cable, and each
of its guests tries it at its own port.

1. Dock the Mac. At ws, list the wired ports with their carrier: `ip -br link` and `ip -br addr`.
2. Set `WS_CABLE_PORT` at ws, and `WS_CABLE_SERVICE` on the Mac. `networksetup
   -listallnetworkservices` lists the services, and the cable's has a `169.254` address.
3. Run the server's bootstrap at ws, as [its runbook](../ubuntu/runbook.md) says, then
   `ws macos setup`.
4. Ping `10.77.0.1`. The Mac's end is `10.77.0.2` ([The cable](README.md#the-cable)).

## What stays by hand

- With Secure Boot on, where the kernel refuses the fans' driver, DKMS's key, enrolled at ws's
  console ([Secure Boot](../ubuntu/roles/fans/README.md#secure-boot)).
- macOS's grant of the local network to an app.
- The server's lease, made static, and the router's forwards where `WS_REMOTE` is set
  ([The router](../ubuntu/roles/cable/router.md)).
- git's signing key on GitHub, and ws's in `@zadykiand`'s settings, which no run adds.
- A log out after the first run, before any app opens, and a reboot. The Dock, Finder, sounds and
  keyboard read their preferences at the next log out, and Logi's installer asks a reboot.
- Apple's sign-in, the language and region, FileVault, the apps' licences and the privacy
  permissions that an app asks for.
- The menu bar, Translate for Safari, and Logi's and Safari's settings.
- GPG's and the YubiKey's keys ([devtools](roles/devtools/README.md)).
- The screen saver, Macintosh, in System Settings › Screen Saver. macOS keeps the choice in a
  file that Apple documents nowhere, so no unit sets it.
- `colima start`, as the machine's size and its start at login are the user's
  ([docker](roles/docker/README.md)).
- The upgrades. Setup installs each cask and formula at its latest and never upgrades it:
  `mise -C /opt/ws/macos -E ws,apps bootstrap packages upgrade` does, as the user. A cask that
  updates itself is left to itself.
