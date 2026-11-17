# The server's runbook

A person does these steps at ws, in this order. The Mac's side sets them in its own order
([The Mac's runbook](../../client/macos/runbook.md)).

1. Log in as the installer's user, by key.
2. Run `install.sh` as root, with `curl -fsSL
   https://github.com/zadykian/ws/releases/latest/download/install.sh | sudo sh`, so that ws lands
   in `/usr/local/bin`, which sudo reaches ([Releases](../../../engine/core/update/README.md)).
   Then set each value that differs from its default with `sudo ws server ubuntu values set KEY
   VALUE`.
3. Make the secrets: `sudo ws server ubuntu secrets init`, then each secret by `sudo ws server
   ubuntu secrets set KIND NAME`. A value never goes in an argument
   ([The secrets](secrets/README.md)).
4. Run `sudo systemd-run --pty --wait -p LoadCredentialEncrypted=ws-secrets ws server ubuntu apply
   --secrets /etc/ws/secrets.age`, as [the secrets file](shell/README.md#the-secrets-file) says.
   A second run changes nothing.
