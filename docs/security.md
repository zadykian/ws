# security

The `security` role sets up an sshd drop-in that lets everyone log in by key only, ufw, and
fail2ban for SSH.

## SSH and the firewall

The playbook turns SSH's passwords off: everyone logs in by key, root included. ufw then lets in
SSH on port 22 and WireGuard on UDP port 51820 ([wireguard](wireguard.md)). It lets in DNS lookups
from Docker's bridges to the host's resolver too ([docker](docker.md)), and nothing else. fail2ban
bans an address for 10 minutes after 5 failed logins in 10 minutes. Over SSH, keep the session that
ran the playbook open until a new one logs in. The playbook stops before it turns passwords off if
root has no key to log in with.

Docker's published ports get past ufw, so Docker publishes a port that names no address on
`127.0.0.1` ([docker](docker.md)): reach it through an SSH tunnel. A port published on another
address, as in `-p 0.0.0.0:8080:80`, is open to every network the host is on.
