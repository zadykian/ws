# The tunnel daemons

`ws client macos tunnel run --server NAME` is the daemon of one server ([The Mac](../README.md)).
Each has its own utun and its own directory, `/etc/ws-tunnel/NAME`, with `tunnel.conf` and
`routes.d`. The Mac's key sits beside the directories. A daemon reads the other servers' directories
too, so that two of them keep apart. It writes nothing there.

## Networks

A network that two servers' `routes.d` hold goes through neither tunnel: each daemon leaves out
what any other holds, by overlap, and logs `leaves NETWORK out: server NAME's routes.d holds
NETWORK` at each read. Neither depends on which reads first.

A network that holds another server's endpoint goes out as one that holds the daemon's own: the
packets of that server's tunnel must not enter this one. The endpoints are the other's
`WS_LAN_ADDRESS` and its `WS_REMOTE`, as the resolver gives it at the start of each round and at
each read.

## Routes

A daemon deletes a route only where `route -n get` says the route goes through its own utun. It
leaves one that goes through another utun or the Mac's own interface alone, and logs `leaves the
route to NETWORK alone: it goes through IFACE, not UTUN`.

## Reading again

An `apply` that writes a server's `tunnel.conf` has every other loaded daemon read again, by
SIGHUP, so that each counts that server's endpoints at once.

`ws client macos tunnel routes` and the removal of a server's daemon, at the next `apply`, have
every loaded daemon read again, by SIGHUP. A daemon routes at once what a gone server held.
