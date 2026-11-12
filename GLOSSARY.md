# ws

ws sets up one development server from code, for one user, `ws_user`, and the Mac that works
on it. These are the words its code and its docs use for the server's side.

## Language

**Account**:
`ws_user` as the server knows it: its name, its home, its uid and its primary gid.
_Avoid_: owner, ids

**Home file**:
A file below the account's home that ws reads or writes as root, by a path under the home. The
account owns it, and no link lies on the way to it.
_Avoid_: user file, plain file

**Pinned file**:
A file of a release that ws fetches at the version a role pins, for the machine's architecture as
dpkg names it. ws checks it against a SHA-256 that the role pins or the step finds as it plans. A
binary is the pinned file at its destination; a package and a tree take one as their source.
_Avoid_: download, release asset, artifact
