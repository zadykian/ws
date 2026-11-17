# The server's runbook

A person does these steps at ws, in this order. The Mac's side sets them in its own order
([The Mac's runbook](../../client/macos/runbook.md)).

1. Log in as the installer's user, by key.
2. Write `/etc/ws/values.yml`, root's and mode 600, with each value that differs from its
   default. `install.sh` reads `WS_USER` from it.
3. Run `install.sh` as root, with `curl -fsSL
   https://github.com/zadykian/ws/releases/latest/download/install.sh | sudo sh`. It installs mise
   and runs its bootstrap, which puts ws in `/usr/local/bin`, which sudo reaches
   ([Releases](../../../engine/core/update/README.md)).
4. Make the secrets: `sudo ws server ubuntu secrets init`, then each secret by `sudo ws server
   ubuntu secrets set KIND NAME`. A value never goes in an argument
   ([The secrets](secrets/README.md)).
5. Run `sudo systemd-run --pty --wait -p LoadCredentialEncrypted=ws-secrets ws server ubuntu apply
   --secrets /etc/ws/secrets.age`, as [the secrets file](shell/README.md#the-secrets-file) says.
   A second run changes nothing.
