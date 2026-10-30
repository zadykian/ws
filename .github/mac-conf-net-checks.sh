# shellcheck shell=sh
# The mac job's checks of how mac/setup.sh's ws_conf reads the network's keys of mac.conf:
# WS_LAN_ADDRESS inside WS_LAN, WS_CABLE_SERVICE and WS_CABLE_PORT. mac-conf-checks.sh sources them
# in their place among its cases, so that they run with its functions: taken, refused and gives.

# WS_LAN_ADDRESS inside WS_LAN, whichever line comes first. A host's address at either end of a /24
# must pass, as must a /31's first. Its network and broadcast addresses, a /30's too, must stop
# ws_conf. So must one far from it, one of the wrong form, and a host's address just above and just
# below a /26, which only its bounds stop.
taken WS_LAN=10.88.0.0/24 WS_LAN_ADDRESS=10.88.0.1
taken WS_LAN_ADDRESS=10.88.0.254 WS_LAN=10.88.0.0/24
taken WS_LAN=10.88.0.0/31 WS_LAN_ADDRESS=10.88.0.0
refused 'inside WS_LAN, 10.88.0.0/24' WS_LAN=10.88.0.0/24 WS_LAN_ADDRESS=10.88.0.0
refused 'inside WS_LAN, 10.88.0.0/24' WS_LAN_ADDRESS=10.88.0.255 WS_LAN=10.88.0.0/24
refused 'inside WS_LAN, 10.88.0.0/30' WS_LAN=10.88.0.0/30 WS_LAN_ADDRESS=10.88.0.3
refused 'inside WS_LAN, 10.88.0.0/24' WS_LAN_ADDRESS=192.0.2.10 WS_LAN=10.88.0.0/24
refused 'inside WS_LAN, 10.88.0.64/26' WS_LAN=10.88.0.64/26 WS_LAN_ADDRESS=10.88.0.129
refused 'inside WS_LAN, 10.88.0.64/26' WS_LAN=10.88.0.64/26 WS_LAN_ADDRESS=10.88.0.62
refused "line 2: WS_LAN_ADDRESS=10.88.0.1x is not a host's IPv4 address inside WS_LAN" \
    WS_LAN=10.88.0.0/24 WS_LAN_ADDRESS=10.88.0.1x

# WS_CABLE_SERVICE, a network service's name. One with blanks must pass in either quotes, and stop
# ws_conf bare, as a shell would run its second word. A tab, or a CR at its end, must stop it too.
gives WS_CABLE_SERVICE 'USB 10/100/1G/2.5G LAN' 'WS_CABLE_SERVICE="USB 10/100/1G/2.5G LAN"'
gives WS_CABLE_SERVICE 'Thunderbolt Ethernet' "WS_CABLE_SERVICE='Thunderbolt Ethernet'"
gives WS_CABLE_SERVICE Ethernet WS_CABLE_SERVICE=Ethernet
refused 'line 1: WS_CABLE_SERVICE=USB LAN is not a network service' 'WS_CABLE_SERVICE=USB LAN'
refused 'is not a network service' "WS_CABLE_SERVICE=\"USB$(printf '\t')LAN\""
refused 'is not a network service' "WS_CABLE_SERVICE=Ethernet$(printf '\r')"
# WS_CABLE_PORT, ws's port of the cable, which goes in a command on ws: a name of Linux's. It may
# stay beside an empty WS_CABLE_SERVICE, as setup.sh's stop tells.
gives WS_CABLE_PORT enp5s0 WS_CABLE_SERVICE=Ethernet WS_CABLE_PORT=enp5s0
gives WS_CABLE_PORT enp5s0 WS_CABLE_PORT=enp5s0 WS_CABLE_SERVICE=
for line in 'WS_CABLE_PORT=enp5s0;id' WS_CABLE_PORT=-enp5s0 WS_CABLE_PORT=enp5s0enp5s0enp5s0; do
    refused "line 1: $line is not $(ws_conf_form WS_CABLE_PORT)" "$line"
done
