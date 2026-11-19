# The Mac's runbook

A person does these steps in this order, as no wizard asks them. Each step is done once the check
beside it holds. Run `MISE_LOCKED=1 mise -C /opt/ws/roles/client bootstrap` where a step says so: it
sets up the Mac's end ([The Mac](README.md)). `MISE_LOCKED=1` makes mise take each tool as the
tree's lock holds it and refuse one that the lock lacks, as CI's `mac` job runs it.

## Tools

1. Install the command line tools with `xcode-select --install`, and wait for it to end.
2. Install ws's binary by hand. Download `ws-darwin-arm64` and `ws.sha256` from
   `releases/latest/download` of ws's repository with `curl -fsSLO`. Check it with `shasum -a 256
   -c --ignore-missing ws.sha256`, install it with `install -m 755 ws-darwin-arm64
   ~/.local/bin/ws`, and put `~/.local/bin` on `PATH`.
3. Install mise 2026.11.9 for [the Mac's frame](macos/frame/README.md):
   - Download `mise-v2026.11.9-macos-arm64` from
     `https://github.com/jdx/mise/releases/download/v2026.11.9/` with `curl -fsSLO`.
   - Check its SHA-256 with `shasum -a 256`. It must be
     `f3bb0e8cbe0217a0b52444ecc0042364563ddd1de0490313bb7e702ecab41a14`.
   - Install it with `sudo install -m 755 mise-v2026.11.9-macos-arm64 /usr/local/bin/mise`.
4. Clone ws's repository at the latest release's tag into `/opt/ws`, as root: `sudo git clone
   --depth 1 --branch vX.Y.Z https://github.com/zadykian/ws /opt/ws`. It stays root's, and no one
   else may write it.
5. Write `/etc/mise/miserc.toml`, as root, with the line `env = ["mac"]`.
6. Link the tree's plugin where mise finds a system plugin, as root: `sudo mkdir -p
   /usr/local/share/mise/plugins && sudo ln -s /opt/ws/plugins/ws /usr/local/share/mise/plugins/ws`
   ([Where it sits](macos/frame/README.md#where-it-sits)).

## Answers

Write the Mac's values in `/etc/ws/values.yml` and each server's in
`/etc/ws/servers/NAME/values.yml`, as root, through sudo ([The two
files](values/README.md#the-two-files)). The files' check refuses a clash.

1. Pick a slot, 0 to 15, that no server has: `WS_SLOT`.
2. Pick the ports: 51820 for the tunnel and 22022 for the router's forward to ws's 22. Where
   `WS_REMOTE` repeats on another server, add the slot to each.
3. Set the rest that differs from its default: `WS_TARGET`, `WS_LAN`, `WS_LAN_ADDRESS`,
   `WS_REMOTE`, `WS_REMOTE_PORT`, `WS_TUNNEL_NETWORKS` and `WS_RESOLVERS`. The
   [router's page](../server/ubuntu/cable/router.md) gives the LAN's defaults.

## The Mac's key

1. Install Secretive from its release's `Secretive.zip`, with no Homebrew. Read the version and
   the SHA-256 once from `https://formulae.brew.sh/api/cask/secretive.json`, its `version` and
   `sha256`. Download the zip from `https://github.com/maxgoedjen/secretive/releases/download/vVERSION/Secretive.zip`,
   check it with `shasum -a 256 Secretive.zip`, unzip it, and move `Secretive.app` to
   `/Applications`. `spctl -a -vv /Applications/Secretive.app` must say `accepted` and `Notarized
   Developer ID`.
2. Open it, and make a P-256 key for the login and one for git's signing. One key may serve both.
   The key stays in the Mac's Secure Enclave.
3. Set `WS_IDENTITY_PUB_KEY`, `WS_IDENTITY_AGENT` and `WS_GIT_SIGNING_PUB_KEY` in
   `/etc/ws/values.yml`, each `.pub` as `/etc/ws/secretive/NAME.pub`. The bootstrap writes those
   copies from the agent ([Secretive's keys](macos/git/README.md#secretives-keys)).

## The bootstrap

Trust the tree once, before the first bootstrap, as the user who runs it:
`cd /opt/ws/roles/client && mise trust --all`. mise asks before it reads the config of a tree that
it does not trust, and the clone is root's, as on the server ([install.sh](../../install.sh)).

Run `MISE_LOCKED=1 mise -C /opt/ws/roles/client bootstrap`, which installs gh and sets up ssh, git,
cld and the Mac's other roles ([The Mac's frame](macos/frame/README.md)). It makes the Mac's
WireGuard key, whose public half is `/etc/ws/tunnel.pub`. A second run changes nothing, and a run
that changes nothing asks no password.

## GitHub

1. Run `gh auth login --scopes write:public_key,write:ssh_signing_key`, at github.com's device flow.
2. Add the key for ws with `gh ssh-key add` and the login key's `.pub`.
3. Add git's signing key with `gh ssh-key add --type signing` and its `.pub`, titled with the
   Mac's name ([The signing key](macos/git/README.md#the-signing-key)).

GitHub is the only source of the server's keys: nothing copies a key to the server. Ubuntu's
installer imports the account's keys for ws's user, so run this step before its install. In the
installer, turn OpenSSH on, import the keys from GitHub and leave password login off.

## The secrets

1. Once, create the secrets repository by `gh repo create zadykian/ws-secrets --private`, with its
   README, license and `check` workflow, whose jobs `values` and `gitleaks` its ruleset on `main`
   requires.
2. With the Nano plugged in, run `ws client macos secrets init`. Where the Nano holds no age
   identity, it asks for the Nano's PIN and a touch. It prints the backup key's secret line once:
   keep it on paper or in a password manager, then press Enter. It lands `fnox.toml` by a pull
   request ([ws client macos secrets init](macos/secrets/README.md#ws-client-macos-secrets-init)).
3. Run it again once `/opt/ws` moves to a release that declares a new input. Until then, a
   server's run of that release fails at mise, which finds no value of the input.

## ws's host key

1. Read ws's key with `ssh-keyscan -t ed25519 ADDRESS`, ADDRESS being ws's LAN address, or the
   router's name and port off the LAN. A new ws has a lease from the router's pool.
2. At ws's console, `ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub` prints ws's own
   fingerprint. Compare it with the scan's, from `ssh-keygen -lf -`.
3. Where they match, append the scan's line to `~/.ssh/known_hosts`, with NAME, the server's name, in
   place of the address. A new ws faces the internet until its `security` role runs.

## Login

Log in with `ssh -o HostName=ADDRESS NAME`, as the tunnel is not up yet. It must log in as `WS_USER`
by the Mac's key. A cloud VM gives its default user the provider's keys, which must be the account's.
Where ssh names another host key, drop the old one with `ssh-keygen -R NAME`.

## Sudo

At ws, `sudo -n true` must pass for `WS_USER`. Where sudo wants a password, type it at the prompt of
a command that needs root. A user outside sudoers cannot go on.

## ws on ws

1. At ws, write the values that differ from their defaults first, as
   [the server's runbook](../server/runbook.md) says. On a cloud VM, set `WS_SECURITY_OUTBOUND`
   to false, as the outbound table drops the metadata service
   ([Outbound](../server/ubuntu/security/README.md#outbound)). At slot N other than 0, set
   `WS_TUNNEL_ADDRESS` to `10.99.N.1/24`, `WS_TUNNEL_PORT` to the Mac's, and
   `WS_TUNNEL_PEERS` to `mac` at `10.99.N.2`
   ([The tunnel](../server/ubuntu/tunnel/README.md#the-tunnel)).
2. Run `install.sh` by the three lines of [Install](../../.github/workflows/release.md#install),
   `set -o pipefail` among them. The bootstrap it runs puts ws in `/usr/local/bin`. Check it with
   `ws version`.
3. Make the secrets, as [the server's runbook](../server/runbook.md) says, with `ssh -t NAME
   sudo ws server ubuntu secrets ...`.
4. Run the server's bootstrap again over `ssh -t NAME`, in the form that
   [the server's runbook](../server/runbook.md) gives. A
   first run takes 20 to 40 minutes. Use `tmux` there: a dropped ssh ends the run.

## claude

Run `ssh -t NAME claude auth login`, and open the address it shows in the Mac's browser.

## Signing key

Copy the server's signing key, which `~/.gitconfig` there names, and add it with `gh ssh-key add
--type signing --title HOSTNAME`. The Mac's gh login holds the scope: ws's token has no account
permission ([Signing](../server/ubuntu/git/README.md#signing)).

## The tunnel

1. Copy the Mac's `/etc/ws/tunnel.pub` to ws's `/etc/wireguard/mac.pub`, as root, and run the
   server's bootstrap there, as [its runbook](../server/runbook.md) says
   ([The tunnel](../server/ubuntu/tunnel/README.md#the-tunnel)).
2. Put wg0's public key, which that page names, in `WS_PUBLIC_KEY` of the server's `values.yml`.
3. Run `MISE_LOCKED=1 mise -C /opt/ws/roles/client bootstrap` again, and allow ws-tunnel in the
   Background.
4. Ping the server's tunnel address, `10.99.N.1` for slot N, and run `ssh NAME`. A handshake must
   come within 30 s.

## The cable

Skip it on a cloud VM and a VM on the Mac.

1. Dock the Mac. At ws, list the wired ports with their carrier: `ip -br link` and `ip -br addr`.
2. Set `WS_CABLE_PORT` at ws, and `WS_CABLE_SERVICE` on the Mac. `networksetup
   -listallnetworkservices` lists the services, and the cable's has a `169.254` address.
3. Run the server's bootstrap at ws, as [its runbook](../server/runbook.md) says, then
   `MISE_LOCKED=1 mise -C /opt/ws/roles/client bootstrap`.
4. Ping `10.77.0.1`. The Mac's end is `10.77.0.2` ([The cable](README.md#the-cable)).

## What stays by hand

- With Secure Boot on, where the kernel refuses the fans' driver, DKMS's key, enrolled at ws's
  console ([Secure Boot](../server/ubuntu/fans/README.md#secure-boot)).
- macOS's grant of the local network to an app.
- The server's lease, made static, and the router's forwards where `WS_REMOTE` is set
  ([The router](../server/ubuntu/cable/router.md)).
- git's signing key on GitHub, which no run adds.
