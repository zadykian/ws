# The wizard's targets

The Mac's wizard sets up ws on one of three machines, which `mac.conf`'s `WS_TARGET` names
([The answers](mac-wizard.md#the-answers)). Each target sets the questions of ws's ways and their
defaults, and what a few stages do. ws's playbook is the same on each, but for the outbound table on
a cloud VM.

## A machine on the LAN

`WS_TARGET=lan`, the default, is a machine at home, behind [The router](router.md), with a cable to
the Mac's display, if any:

- `WS_LAN` is the home LAN, `10.88.0.0/24` by default, and `WS_LAN_ADDRESS` ws's static lease
  there, `10.88.0.10`.
- `WS_REMOTE` is the router's name on the internet, if any, and `WS_REMOTE_PORT` its forward to
  ws's 22, `22022` by default.
- Where ws doesn't answer at its way, the Host key stage asks its address as the router's leases
  show it.
- The Cable stage sets up the cable ([The cable](mac-wizard.md#the-cable)).
- The closing list names ws's lease and the router's forwards.

## A cloud VM

`WS_TARGET=cloud` is a VM at a provider, reached through the internet alone:

- The wizard asks no LAN.
- `WS_REMOTE` is the VM's name or address, and `WS_REMOTE_PORT` its SSH port, `22` by default. The
  target needs both.
- The VM needs a public IPv4 address, as `ws-ssh` and the tunnel take IPv4 alone.
- The provider's firewall must let in TCP `WS_REMOTE_PORT`, and UDP 51820, the tunnel's, or the port
  that `WS_PORT` sets. The closing list says so.
- The VM's default user takes the provider's keys, which must be the GitHub account's
  ([Root's key](mac-wizard.md#roots-key)).
- Where ws doesn't answer at its way, the Host key stage asks the VM's public address, as the
  provider shows it.
- ws's run turns the outbound table off, which would drop the metadata service
  ([ws's run](mac-wizard.md#wss-run)).
- The wizard skips the Cable stage.

## A VM on the Mac

`WS_TARGET=mac-vm` is a VM that the Mac runs. The Mac is its gateway, DHCP server and DNS:

- `WS_LAN` is the Mac's network for its VMs, `192.168.64.0/24` by default, the shared network of
  macOS's Virtualization framework. Parallels uses `10.211.55.0/24`, and VMware Fusion a
  `172.16.N.0/24` of its own. At the VM's console, `ip -4 route` shows the Mac as its gateway there.
- `WS_LAN_ADDRESS` is the VM's address there, which the target needs, as `ip -4 address` prints it.
  Keep it stable, by a static address in the VM or a reservation in its app's DHCP: else ssh goes
  to another host once the lease moves.
- ssh reaches the VM there while the Mac has an address on that network, as while a VM runs
  ([ssh's hosts](mac.md#sshs-hosts)). Else the Host key stage stops, and says to start the VM.
- The wizard asks no `WS_REMOTE`, as only the Mac reaches the VM.
- The outbound table stays on: ws's DHCP and DNS go to the Mac, which the table lets through
  ([Outbound](security.md#outbound)).
- The wizard skips the Cable stage.

## A move to another target

To move `mac.conf` to another target, set `WS_TARGET`, and remove every key of ws's ways: `WS_LAN`,
`WS_LAN_ADDRESS`, `WS_REMOTE`, `WS_REMOTE_PORT`, `WS_CABLE_SERVICE` and `WS_CABLE_PORT`. An old way
would lead to the old machine still. From the LAN to a cloud VM, the router's `WS_REMOTE` would, and
the cloud VM's run would turn that machine's outbound table off. `setup.sh` refuses a key of another
target's ways: `WS_LAN` on a cloud VM, `WS_REMOTE` on a VM on the Mac, and `WS_CABLE_SERVICE` or
`WS_CABLE_PORT` off the LAN. Then run the wizard, which asks the new target's ways. A new machine
offers another host key, so run `ssh-keygen -R ws` first. From a cloud VM, remove
`security_outbound: false` from ws's `host_vars` too, as the closing list says, then run the wizard
with `--update`.
