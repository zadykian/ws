# The tunnel daemons

`ws macos tunnel run --server NAME` is the daemon of one server ([The Mac](../../README.md)).
Each has its own utun and its own directory, `/etc/ws/tunnel/NAME`, with `tunnel.conf` and
`routes.d`. The Mac's key sits beside the directories. A daemon reads the other servers' directories
too, so that two of them keep apart. It writes nothing there.

## Networks

A network that two servers' `routes.d` hold goes through neither tunnel: each daemon leaves out
what any other holds, by overlap, and logs `leaves NETWORK out: server NAME's routes.d holds
NETWORK` at each read. Neither depends on which reads first.

A network that holds another server's endpoint goes out as one that holds the daemon's own: the
packets of that server's tunnel must not enter this one. The endpoints are the other's
`WS_LAN_ADDRESS` and its `WS_REMOTE`, as the resolver gives it at the start of each round and at
each read. The daemon looks up the other servers' remotes at once, with 2 s for them all. One
with no answer by then counts as unresolved, as one that fails does, until the next round or read.

## Routes

A daemon deletes a route only where `route -n get` names the network itself, by its destination
and mask, and says the route goes through its own utun. A wider route or the default that answers
in its place counts as gone. A /32 counts with no mask line, as a host route's answer may hold
none. A daemon leaves a route that goes through another utun or the Mac's own interface alone, and
logs `leaves the route to NETWORK alone: it goes through IFACE, not UTUN`.

## A yield

`/var/run/ws-tunnel.yield`, root's, holds the networks that another owner routes in place of the
tunnels, a network a line. Each daemon reads it at its start and at each read. It leaves out each
of its networks that a yielded one holds, equal or narrower, and logs `leaves NETWORK out, as it
yields YIELDED to another owner`. It deletes that network's route by the rule above. A wider
network stays routed, as the other owner's narrower routes win by length. Once the file is gone, a
read routes the networks again.

A missing file yields nothing, and a daemon skips a line that is no IPv4 network. A reboot clears
the file. `ws macos tunnel status` prints the yield that the routes took, as `yielded to another
owner: NETWORKS`.

## Reading again

Each daemon's job in launchd's unit of the tunnel lists every server's `tunnel.conf` and
`routes.d`. So a bootstrap that changes one sends every loaded daemon a SIGHUP, and each counts
that server's endpoints and networks at once.

`ws macos tunnel routes`, and the removal of a server's daemon by `ws macos tunnel
conf`, have every loaded daemon read again, by SIGHUP. A daemon routes at once what a gone server
held.

## The cable

The cable reaches one server, the one whose file has `WS_CABLE_SERVICE`, and its guests. Their
`tunnel.conf` says `WS_CABLE=yes`, and only their daemons and the server's direct way try
`10.77.0.1`, first, each at its own port. The other servers leave the cable out, as they would try
an address that is no server of theirs.

## A new copy of ws

A daemon looks at root's copy of `ws`, `/usr/local/bin/ws`, at each tick, 10 s after the last.
Where a new file took its place, as install.sh does, it logs `root's copy of
ws is new; stops, for launchd to start the new one`, and launchd starts the new copy.
