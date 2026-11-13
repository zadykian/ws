# ws

ws sets up one development server from code, for one user, `ws_user`, and the Mac that works
on it. These are the words its code and its docs use, for the server's side and for the Mac's.

## Language

**Account**:
`ws_user` as the server knows it: its name, its home, its uid and its primary gid.
_Avoid_: owner, ids

**Entry**:
One value of the secrets file, by its kind and its name: a token, an env variable, a git server's
credential, a profile file or a password.
_Avoid_: secret, field

**Home file**:
A file below the account's home that ws reads or writes as root, by a path under the home. The
account owns it, and no link lies on the way to it.
_Avoid_: user file, plain file

**Identity's credential**:
ws's age identity as the systemd credential `/etc/credstore.encrypted/ws-secrets`, which the
host's key encrypts. It alone opens the secrets file, and a unit loads it by
`LoadCredentialEncrypted=ws-secrets`.
_Avoid_: age key, key file

**Label**:
An entry's name as the commands and the forms give it, `KIND NAME`, as `token github`. `keys`
prints one a line; a label never holds a value.
_Avoid_: field name

**Pinned file**:
A file of a release that ws fetches at the version a role pins, for the machine's architecture as
dpkg names it. ws checks it against a SHA-256 that the role pins or the step finds as it plans. A
binary is the pinned file at its destination; a package and a tree take one as their source.
_Avoid_: download, release asset, artifact

**Secrets file**:
`/etc/ws/secrets.age` on ws, root's and 600: TOML that age encrypts to the identity's credential,
a table per kind. `secrets.Store` alone reads and writes it.
_Avoid_: vault

## The Mac's side

**Named key**:
The key whose `.pub` `mac.conf` names for a use: ws's login, by `WS_IDENTITY_PUB_KEY`, or git's
signing, by `WS_GIT_SIGNING_PUB_KEY`. Either ws's copy of a key of Secretive's agent, or a file
key.
_Avoid_: picked key, git key

**Operation**:
One job that the wizard asks of ws, a method of the session. It sends an op of the remote or the
files script, or two in turn, as naming the cable's port and then running its roles. Its row in
the session's table says whether it changes ws, so that the next status reads ws again.
_Avoid_: command, request

**Session**:
The wizard's one way to ws: from the Mac over ssh, root by sudo, or ws's own on the machine. A
stage asks it each operation. It holds what ws's status read until an operation changes ws.
_Avoid_: remote, connection

**User's directory**:
A directory below the Mac's home that a step writes under, such as `~/.ssh`. A run makes it where
missing, and stops where the path is no directory. It leaves the mode of one that exists, but for
ws's directory and ssh's `config.d`. Core's `file.UserDir` holds it.
_Avoid_: folder

**User's file**:
A whole file below the Mac's home that ws writes as the user, not root, such as
`~/.config/ws/gitconfig`. A plan and a report read it alike. Core's `file.UserFile` holds it. On the
server, root writes a home file.
_Avoid_: dotfile

**User's lines**:
ws's lines in a user's file that holds other text too, such as `Include config.d/ws` at the top of
`~/.ssh/config`. A run adds them once, at the top or the end, and keeps the old file as
`PATH.TIME~`. At a link that lacks them, or lines out of place, the step warns or stops.
Core's `file.UserLines` hold them.
_Avoid_: include line

**ws's directory**:
`~/.config/ws` on the Mac, mode 700, which holds `mac.conf`, `servers/` and the files that git, zsh
and the tunnel read. A run sets that mode where it differs.
_Avoid_: config directory
