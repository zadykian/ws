#!/bin/sh
# Written by github.com/zadykian/ws, whose next run replaces any change made here.
#
# Drops what Docker's containers send to the Mac through the interfaces given, wg0 and the cable,
# but for replies to the Mac's own connections. wireguard-docker-user.service runs it:
#
#   docker-user.sh INTERFACE...
#
# The drops come from docker0 and the br- bridge of each of Docker's networks, for IPv4 and IPv6,
# in a chain of their own, WS-MAC, that DOCKER-USER jumps to first. Docker's iptables backend runs
# DOCKER-USER ahead of its own rules, makes the chain only where missing, and never changes what it
# holds, so the drops last through Docker's restarts.
#
# Each run fills WS-MAC anew, so that an interface no longer given loses its drops. It does so in
# one commit of iptables-restore, whose chain line empties WS-MAC under --noflush. Rule by rule, a
# container's connection to the Mac opened while WS-MAC lacked its drops would pass as ESTABLISHED
# once they were back.

set -eu

drop='-m conntrack ! --ctstate ESTABLISHED,RELATED -j DROP'
for ipt in iptables ip6tables; do
    "$ipt" -w -n -L DOCKER-USER >/dev/null 2>&1 || "$ipt" -w -N DOCKER-USER
    {
        printf '%s\n' '*filter' ':WS-MAC - [0:0]'
        for out; do
            for bridge in docker0 br-+; do
                printf '%s\n' "-A WS-MAC -i $bridge -o $out $drop"
            done
        done
        printf '%s\n' COMMIT
    } | "$ipt"-restore -w --noflush
    "$ipt" -w -C DOCKER-USER -j WS-MAC 2>/dev/null || "$ipt" -w -I DOCKER-USER -j WS-MAC
done
