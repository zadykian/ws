# The YubiKey

The Mac's key for ws and git's signing key live in the PIV applet of a YubiKey 5C Nano. Its firmware
is 5.7 or later, which makes Ed25519 keys. `mac/setup.sh` finds each key by name on the one YubiKey
in, makes it where missing, and writes its public key in `~/.config/ws/keys` ([The Mac](mac.md)).

## mac.conf's names

`WS_SSH_KEY` names the key that ssh offers ws, `ws` by default, and `WS_GIT_KEY` the one that signs
git's commits, `git` by default. A name is 1 to 32 of `a-z`, `0-9` and `-`, and the two differ, as
each key has one use. A key's name is the subject of its slot's certificate, `CN=NAME`, and its
public key goes in `~/.config/ws/keys/NAME.pub`. The wizard asks neither name.

## The keys

`setup.sh` installs Homebrew's ykman where `PATH` has none. It installs Homebrew's yubico-piv-tool,
whose libykcs11 gives ssh's agent the keys, and terminal-notifier, for [The notices](#the-notices),
where Homebrew lacks them. `--check` names the install. Where terminal-notifier's install fails,
the script warns and goes on, as that costs the notices alone. It stops unless one YubiKey is in,
and names each by its serial where several are, as a second in the dock would be. Each later call
of ykman names the one by `--device SERIAL`.

A key is the slot whose certificate is `CN=NAME` in `ykman piv info`, in any slot. Two slots of one
name stop the script. The script exports the key with `ykman piv keys export`, and writes it as
`ssh-ed25519 KEY NAME@yubikey`, mode 644, as `ssh-keygen -i` takes no Ed25519 key. Where the file
held another key, the script warns: another YubiKey's, or the same one's before a reset of its PIV
applet. GitHub and ws need the new key, as they hold the old one alone.

A key must be Ed25519, as ws makes and tests no other, with the PIN policy `NEVER` and the touch
policy `NEVER`, as `ykman piv keys info SLOT` shows them. ssh and git use it unattended, with no PIN
typed: in a rebase, and in Root's login, which asks nothing. The script stops on a key with another
type or policy, and on a certificate without its key, naming the commands that free the slot:

```sh
ykman piv keys delete 82
ykman piv certificates delete 82
```

The next run then makes the key again, and warns of its new public key. While the YubiKey is in, any
program that runs as the user can sign with the keys, as with the file keys before them.

## Making a key

A missing key goes in the first retired slot, of 82 to 95, that holds neither a key nor a
certificate. `setup.sh` runs ykman twice at the terminal, with the slot and a temporary file:

```sh
ykman piv keys generate -a ED25519 --pin-policy NEVER --touch-policy NEVER 82 ws.pem
ykman piv certificates generate -s CN=ws -d 3650 82 ws.pem
```

The certificate is valid for 3650 days, not ykman's 365. ssh and git use the key alone and ignore
the certificate's expiry. Other readers of the PIV certificate, such as macOS's smart-card pairing
or Yubico Authenticator, show an expired one, and 10 years outlast the key.

The first asks the management key, blank for the default. The second asks it again, then the PIN. A
management key that the PIN protects, as `--protect` sets it, takes the PIN instead. With no
terminal, the script stops, naming both commands, which make the key by hand too. `--check` asks
nothing, and names each key it would make and its slot. A YubiKey of firmware older than 5.7 stops
the script, as it makes no Ed25519 key. Where the certificate fails, as on a mistyped PIN, the key
stays in its slot with no name, and a next run would take the next slot. The script stops, naming
the certificate's step alone,
`ykman --device SERIAL piv certificates generate -s CN=NAME -d 3650 SLOT`, which names the key, and
`ykman piv keys delete SLOT`, which frees the slot.

## The PIV applet's defaults

A YubiKey comes with the PIN `123456`, the PUK `12345678` and a management key that anyone knows.
With them, any program on the Mac could replace or delete the keys. `setup.sh` warns at each run
where `ykman piv info` reports a default, and changes none, as a PUK it chose would be lost to the
user. Set them once, by hand:

```sh
ykman piv access change-pin
ykman piv access change-puk
ykman piv access change-management-key --protect
```

The last keeps a new, random management key on the YubiKey, behind the PIN, so that ykman asks the
PIN alone.

## ws's agent

ssh and git sign through an agent of ws's own, `mac/ws-agent`. macOS's own agent takes no `-P`, and
its default providers, in `/usr/lib` and `/usr/local/lib`, leave out Homebrew's libykcs11, which
lies in the Cellar. `ws-agent` runs Apple's `/usr/bin/ssh-agent` at `~/.ssh/ws-agent.sock`, with
libykcs11's path in the Cellar as its one provider. It loads the keys by `ssh-add -c -s`, with the
PIN `VERIFY_NONE` from `ws-agent-pin`: libykcs11 then verifies no PIN, and the agent stores none.

`setup.sh` copies both, and `ws-agent-confirm`, to `~/.config/ws`. Its LaunchAgent,
`~/Library/LaunchAgents/com.github.zadykian.ws-agent.plist`, has launchd run the copy in `gui/UID`
from login on, and log to `~/Library/Logs/ws-agent.log`. A run that changes a copy or the plist
starts the agent again. So does one where the agent lacks a key of `~/.config/ws/keys`, as after
`setup.sh` made one. It then waits up to 10 s for the keys, and warns where one is missing still.
With no login at the screen, as over ssh alone, launchd loads the agent at the next login. macOS
lists it under System Settings › General › Login Items & Extensions, where "Allow in the
Background" must stay on, or `setup.sh` stops.

- It holds the keys only while the YubiKey is in. It reads `ioreg` every 5 s, takes the keys out
  when the YubiKey goes, and loads them again after a plug or a replug. No signature tests them.
- While the YubiKey is in and the agent lists no key, as after a failed load or `ssh-add -D`, it
  loads them again. It tries every 5 s for a minute, then every 30 s, and logs a failure once.
- With a second YubiKey in, it holds the keys of both: `ssh` and git pick theirs by its `.pub`.
- After a `brew upgrade yubico-piv-tool`, which moves libykcs11 in the Cellar, it starts again.

To list the keys it holds:

```sh
SSH_AUTH_SOCK=~/.ssh/ws-agent.sock ssh-add -L
```

`WS_AGENT_SOCKET` gives it another socket, so that a probe by hand runs beside it. Apple's agent
must load Homebrew's libykcs11, then list the keys and sign with them. Only where library validation
refused the library would the agent become Homebrew's openssh's:

```sh
WS_AGENT_SOCKET=/tmp/probe.sock ~/repository/ws/mac/ws-agent &
SSH_AUTH_SOCK=/tmp/probe.sock ssh-add -L
SSH_AUTH_SOCK=/tmp/probe.sock ssh-keygen -Y sign -f ~/.config/ws/keys/git.pub -n git FILE
kill %1
```

## The notices

The agent runs `ws-agent-confirm` at each signature, as `ssh-add -c` asks. It posts a notice through
terminal-notifier, and allows the signature.

- A notice names the key: `ssh key NAME` for `WS_SSH_KEY`'s, and `git key NAME` for `WS_GIT_KEY`'s.
  The plist gives the agent both names. Another key of the YubiKey's goes by its label in
  libykcs11, as `key Public key for Retired Key 3`.
- A key posts once in 5 s at most. Its next notice counts the signatures since the last, and
  replaces it. So an ssh login posts one notice, and a rebase of many commits a few.
- A first run of the wizard opens about 20 connections to ws. Each that comes 5 s or more after the
  last posts a notice.
- `~/.config/ws/stamps` holds a stamp per key: the time of its last notice, and the uses since.
  Beside it, `ID.pid` names the key's last notifier. While that runs, as terminal-notifier does
  until its first prompt is answered, the key starts no other, and its uses fold into a later notice.
- Without terminal-notifier, or with its notices off, each signature passes with no notice. Without
  `ws-agent-confirm`, the agent refuses each signature, until `setup.sh` copies it again.
- Where macOS's `/bin/sh` is dash, which closes no descriptor above 9, the agent can wait for the
  notifier to exit. Each signature of each client then waits: at the first notice, until its prompt
  is answered.

By hand, once:

- Allow terminal-notifier's notices at its first prompt, which the first signature brings.
- In System Settings › Notifications › terminal-notifier, choose Banners or Alerts. Focus and
  Scheduled Summary can hold the notices back.
