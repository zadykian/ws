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
