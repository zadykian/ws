# The router

The router, a MikroTik on RouterOS 7, takes these steps once. Take each in Safe Mode, WinBox's
button or Ctrl+X in its terminal, which undoes the changes if the session drops:

- The LAN on `10.88.0.0/24`, clear of the cable's `10.77.0.0/30`, the tunnel's `10.99.0.0/24` and
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

  Clients move to the new range as they next renew their lease.
- The server's lease, made static at `10.88.0.10`, HOST being the host name its lease shows:

  ```routeros
  /ip dhcp-server lease make-static [find host-name=HOST]
  /ip dhcp-server lease set [find host-name=HOST] address=10.88.0.10
  ```

- Two forwards from the internet to the server: UDP 51820, the tunnel's, and TCP 22022 to its SSH,
  for networks that block UDP. The default firewall's rule that drops what comes in from the WAN
  without a dst-nat lets both through:

  ```routeros
  /ip firewall nat add chain=dstnat in-interface-list=WAN protocol=udp dst-port=51820 \
      action=dst-nat to-addresses=10.88.0.10 to-ports=51820
  /ip firewall nat add chain=dstnat in-interface-list=WAN protocol=tcp dst-port=22022 \
      action=dst-nat to-addresses=10.88.0.10 to-ports=22
  ```

- A name for the router's public address: `/ip cloud set ddns-enabled=yes`, as `auto`, the default
  since RouterOS 7.17, enables it only with Back To Home. `/ip cloud print` shows its `dns-name`,
  `SERIAL.sn.mynetname.net`, for a CNAME at your domain's DNS; its records live 60 seconds. Its
  `public-address` must be the WAN's own: behind the ISP's NAT, neither forward reaches the router.
