# secrets

The `secrets` role installs fnox, which decrypts ws's secrets on the server. It has no spec and no
unit: its `mise.server.toml` pins fnox alone.

## fnox

mise installs fnox 1.50.0 from jdx/fnox's release on GitHub, as `[tools]` pins it, before any unit
of ws. [mise.server.lock](../../mise/mise.server.lock) holds the url of the asset for `linux-x64`
and `linux-arm64`, with GitHub's digest of each and the release's sigstore signer. A wrong sum
fails the run, and `MISE_LOCKED=1` refuses a fnox that the lock lacks. `go test` fails where the
pin, the lock and the digests that `pins_test.go` holds disagree.

- To bump fnox, change its version in `[tools]`, and the sums in `pins_test.go` to GitHub's
  digests of the new assets. Then write the lock again in a container with git and mise:
  `MISE_ENV=server mise -C roles/server lock --platform linux-x64,linux-arm64`, which writes
  `roles/server/mise/mise.server.lock`.
- The Mac pins the same version of fnox, as both sides read one repository of secrets.
