# ikeward

The `ikeward` role sets the server up for an IKEv2 VPN with
[ikeward](https://github.com/zadykian/ikeward), from a client profile that hwdsl2's `ikev2setup.sh`
makes. Without a profile, it does nothing, as in CI's container.

## The profile

`ikeward_profile` names the two files of the profile that ikeward reads, `NAME.p12` and
`NAME.sswan`, by an absolute path less the extension. They hold the client's key, so they stay out
of this repository, on the server:

```sh
/root/repository/ws/bootstrap.sh -e ikeward_profile=/root/vpn/vpnclient
```

## A run

`group_vars/all.yml` pins ikeward's version, and the SHA-256 of its binary for each architecture,
as the release's `ikeward.sha256` gives it. The run downloads the binary into
`/usr/local/sbin/ikeward`, where ikeward's `install.sh` would put it, checked against the pin. A new
pin replaces it.

## By hand

The role never removes ikeward. Without `ikeward_profile`, it leaves the binary alone. ikeward's
[Stop and uninstall](https://github.com/zadykian/ikeward/blob/v0.3.0/README.md#stop-and-uninstall)
says how to remove it. The VPN's server, and the profile `ikev2setup.sh` makes there, are set up by
hand.
