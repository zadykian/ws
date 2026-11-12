# ws

ws sets up one development server from code, for one user, `ws_user`, and the Mac that works
on it. These are the words its code and its docs use for the server's side.

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
