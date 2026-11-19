# ws

ws sets up one development server from code, for one user, `WS_USER`, and the Mac that works
on it. These are the words its code and its docs use, for the server's side and for the Mac's.

## Language

**Account**:
`WS_USER` as the server knows it: its name, its home, its uid and its primary gid.
_Avoid_: owner, ids

**Entry**:
One value of the secrets file, by its kind and its name: a token, an env variable, a git server's
credential, a profile file or a password.
_Avoid_: secret, field

**Frame**:
What each role of mise's config leans on: `mise.toml` with ws's pin, the server's
`mise.server.toml`, the plugin `plugins/ws`, the machine's miserc, and the role `frame`
([The frame](roles/server/ubuntu/frame/README.md)). The plugin's link puts it where mise finds a
system plugin.
_Avoid_: skeleton, scaffold

**Home file**:
A file below the account's home that ws reads or writes as root, by a path under the home. The
account owns it, and no link lies on the way to it.
_Avoid_: user file, plain file

**Identity's credential**:
ws's age identity as the systemd credential `/etc/credstore.encrypted/ws-secrets`, which the
host's key encrypts. It alone opens the secrets file, and `ws unit` decrypts it by `systemd-creds
decrypt`.
_Avoid_: age key, key file

**Kind**:
A resource type that ws writes once in Go, for what mise's own parts lack, such as `file` or
`deb`. It names no role's word ([kind](engine/core/kind/README.md)).
_Avoid_: plugin, module

**Label**:
An entry's name as the commands and the forms give it, `KIND NAME`, as `token github`. `keys`
prints one a line; a label never holds a value.
_Avoid_: field name

**Pinned file**:
A file of a release that ws fetches at the version a role pins, for the machine's architecture as
dpkg names it. ws checks it against a SHA-256 that the role pins or the unit finds as it plans. A
binary is the pinned file at its destination; a package and a tree take one as their source.
_Avoid_: download, release asset, artifact

**Spec**:
A role's `units.toml.tmpl`: TOML over the machine's values that names the role's units, a table a kind
and a key a unit.
_Avoid_: manifest

**Secrets file**:
`/etc/ws/secrets.age` on ws, root's and 600: TOML that age encrypts to the identity's credential,
a table per kind. `secrets.Store` alone reads and writes it.
_Avoid_: vault

**Unit**:
One resource of a kind, named `KIND/NAME`, as `file/sshd`, that a spec names. It plans and applies
as a step does, which `ws unit` runs for mise ([ws unit](engine/core/kind/README.md#ws-unit)).
_Avoid_: package, step

## The Mac's side

**Named key**:
The key whose `.pub` `values.yml` names for a use: ws's login, by `WS_IDENTITY_PUB_KEY`, or git's
signing, by `WS_GIT_SIGNING_PUB_KEY`. Either ws's copy of a key of Secretive's agent, or a file
key.
_Avoid_: picked key, git key

**User's directory**:
A directory below the Mac's home that a unit writes under, such as `~/.local/bin`. A run makes it
where missing, and stops where the path is no directory. It leaves the mode of one that exists. Core's
`file.UserDir` holds it.
_Avoid_: folder

**User's file**:
A whole file below the Mac's home that ws writes as the user, not root, such as Zed's
`~/.config/zed/global_settings.json`. A plan and a report read it alike. Core's `file.UserFile`
holds it. On the server, root writes a home file.
_Avoid_: dotfile

**User's lines**: ws's lines in a user's file that holds other text too, such as `Include
/etc/ws/ssh_config` at the top of `~/.ssh/config`. A run adds them once, at the top or the end, and
keeps the old file as `PATH.TIME~`. At a link that lacks them, or lines out of place, the unit warns
or stops. Core's `file.UserLines` hold them on the Mac, and `file.HomeLines`, read and written as
root, on ws. _Avoid_: include line

**server's name**:
What `--server` takes, and the directory `servers/NAME/` of the server's file: a lower-case letter,
then up to 11 lower-case letters, digits and `-`, other than `direct`, `first` and `key`.

**slot**:
A server's number from 0 to 15, kept as `WS_SLOT` in its file. The maintainer picks the smallest free
slot; the values' check refuses a clash. Slot N's tunnel is `10.99.N.0/24`, inside
`10.99.0.0/20`, with the server at `.1` and the Mac at `.2`.

**ws's directory**:
`/etc/ws`, root's and 755 on both sides, holds git's `gitconfig` and `allowed_signers`. On the Mac
it also holds `values.yml` and `servers/`, which only sudo writes, and the script
`secretive-ssh-keygen`. On ws it holds git's tokens, the env file and the fans' config.
_Avoid_: config directory
