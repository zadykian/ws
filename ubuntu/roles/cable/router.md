# The router

The router, a MikroTik on RouterOS 7, takes these steps once. Take each in Safe Mode, WinBox's
button or Ctrl+X in its terminal, which undoes the changes if the session drops:

- The LAN on `10.88.0.0/24`, clear of the cable's `10.77.0.0/30`, the tunnels' `10.99.0.0/20` and
  Docker's `172.16.0.0/12`. It stays off RouterOS's default, `192.168.88.0/24`, which cafés and
  other homes use too. Run these from WinBox connected by MAC address, which the change doesn't
  drop, once `/export` has shown the names these commands find:

  ```routeros
  /ip address set [find address="192.168.88.1/24"] address=10.88.0.1/24
  /ip pool set default-dhcp ranges=10.88.0.100-10.88.0.254
  /ip dhcp-server network set [find address="192.168.88.0/24"] address=10.88.0.0/24 \
      gateway=10.88.0.1 dns-server=10.88.0.1
  /ip dns static set [find name=router.lan] address=10.88.0.1
  ```

  A client keeps its lease of the old range, whose gateway no longer answers, as its renewal goes
  to the router's old address. It moves only once it rebinds, at 7/8 of the lease time, or
  reconnects. So reconnect the other clients, and have ws take a new lease, from its console or
  over the cable, UPLINK being its port to the router, as `ip route` names it:

  ```sh
  networkctl reconfigure UPLINK
  ```

  Or, a whole lease time ahead of the change, shorten the DHCP server's lease, so that every client
  rebinds within minutes, once `print` has shown the lease time to restore:

  ```routeros
  /ip dhcp-server print
  /ip dhcp-server set [find] lease-time=5m
  ```

  Once every client has moved, set it back, here to RouterOS 7's default of 30 minutes:

  ```routeros
  /ip dhcp-server set [find] lease-time=30m
  ```

- A segment of its own for the server's port, `10.88.1.0/24`, with the router at `10.88.1.1`. On
  the LAN's bridge, root on the server could forge its source address and answer ARP for the
  other hosts, so no rule on its address would bind it. Off the bridge, the interface guards it.
  PORT is the router's port to the server, `ether5` say. Run these from another port, or from
  WinBox connected by MAC address, as the port's link drops. The list `SERVER` stays outside
  `LAN`:

  ```routeros
  /interface bridge port remove [find interface=PORT]
  /interface list add name=SERVER
  /interface list member add list=SERVER interface=PORT
  /ip address add interface=PORT address=10.88.1.1/24
  /ip pool add name=server ranges=10.88.1.100-10.88.1.254
  /ip dhcp-server add name=server interface=PORT address-pool=server lease-time=30m \
      disabled=no
  /ip dhcp-server network add address=10.88.1.0/24 gateway=10.88.1.1 dns-server=10.88.1.1
  ```

  The Mac's `WS_LAN` is then `10.88.0.0/23`, which holds both networks, and its `WS_LAN_ADDRESS`
  `10.88.1.10` ([values](../../../macos/values/README.md#a-machine-on-the-lan)).
- Input from `SERVER`: DNS and DHCP, ahead of the default firewall's drop of what comes from
  outside `LAN`. That drop then guards the router's other addresses and its services, its public
  address too, which ws's table takes for the internet
  ([Outbound](../security/README.md#outbound)):

  ```routeros
  /ip firewall filter add chain=input in-interface-list=SERVER protocol=udp dst-port=67 \
      action=accept place-before=([find where chain=input dynamic=no]->0)
  /ip firewall filter add chain=input in-interface-list=SERVER dst-address=10.88.1.1 \
      protocol=udp dst-port=53 action=accept place-before=([find where chain=input dynamic=no]->0)
  /ip firewall filter add chain=input in-interface-list=SERVER dst-address=10.88.1.1 \
      protocol=tcp dst-port=53 action=accept place-before=([find where chain=input dynamic=no]->0)
  ```

  Each needs a static rule in the input chain to go before, such as the default firewall's: on a
  chain with none, `place-before` fails with "no such item".
- Layer 2: MAC-Winbox, MAC-Telnet and neighbour discovery stay off `SERVER`. The IP filter never
  sees them, so their lists keep them out. RouterOS 7's default configuration sets each to the
  list `LAN`. Print each, and check that it names `LAN` and not `all` or `SERVER`:

  ```routeros
  /tool mac-server print
  /tool mac-server mac-winbox print
  /ip neighbor discovery-settings print
  ```

  Where one does not, set its `allowed-interface-list`, or `discover-interface-list`, to `LAN`.
- Forward: what `SERVER` starts goes to the `WAN` alone, so it reaches no host of the LAN. The
  replies to what the LAN starts pass, as the default firewall accepts the established. Reverse
  path filtering is strict, so that a packet from `SERVER` with a LAN source address, which a
  route back would send to the bridge, drops:

  ```routeros
  /ip firewall filter add chain=forward connection-state=new in-interface-list=SERVER \
      out-interface-list=!WAN action=drop
  /ip settings set rp-filter=strict
  ```

  A router with two uplinks, whose replies may leave by the other one, can't keep the filter
  strict.
- The server's lease, made static at `10.88.1.10`, HOST being the host name its lease shows. ws
  holds a lease on the LAN too, from the first step, so the commands pick the lease of the DHCP
  server `server`. Have ws take a lease from it, from its console or over the cable, UPLINK being
  its port to the router, as `ip route` names it:

  ```sh
  networkctl reconfigure UPLINK
  ```

  ```routeros
  /ip dhcp-server lease make-static [find host-name=HOST server=server]
  /ip dhcp-server lease set [find host-name=HOST server=server] address=10.88.1.10
  ```

  ws takes the address as it next renews, or at once with `networkctl reconfigure UPLINK` again.
- Two forwards from the internet to the server: UDP 51820, the tunnel's, and TCP 22022 to its SSH,
  for networks that block UDP. The default firewall's rule that drops what comes in from the WAN
  without a dst-nat lets both through:

  ```routeros
  /ip firewall nat add chain=dstnat in-interface-list=WAN protocol=udp dst-port=51820 \
      action=dst-nat to-addresses=10.88.1.10 to-ports=51820
  /ip firewall nat add chain=dstnat in-interface-list=WAN protocol=tcp dst-port=22022 \
      action=dst-nat to-addresses=10.88.1.10 to-ports=22
  ```

- A name for the router's public address: `/ip cloud set ddns-enabled=yes`, as `auto`, the default
  since RouterOS 7.17, enables it only with Back To Home. `/ip cloud print` shows its `dns-name`,
  `SERIAL.sn.mynetname.net`, for a CNAME at your domain's DNS; its records live 60 seconds. Its
  `public-address` must be the WAN's own: behind the ISP's NAT, neither forward reaches the router.
- What the steps miss. A cloud VM has no router of yours, so its provider's network isolates it.
  A VM on the Mac reaches the router from the Mac's address, through the Mac's NAT, so it shares
  the LAN's segment with the Mac. The steps are IPv4's. Give `SERVER` no IPv6 prefix, so that
  ws has no IPv6 route to the router's addresses, which its table would pass as routed.
