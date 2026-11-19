#!/bin/bash
# The secrets' tools that mise bootstrap (28-bootstrap.sh) copied into /usr/local/bin: fnox and
# age-plugin-yubikey must be mise's, root's and 0755, at their pins. Then fnox must encrypt a value
# to the Nano's recipient with no YubiKey, through the plugin on PATH. An SSH key that is a
# recipient too must read it back, as a server's host key does.

set -eo pipefail

fail=0
# error MESSAGE... prints an ::error:: line, and fails the case at its end.
error() {
    echo "::error::$*"
    fail=1
}

for tool in fnox age-plugin-yubikey; do
    from=$(cd / && MISE_LOCKED=1 MISE_TRUSTED_CONFIG_PATHS=/opt/ws mise -C /opt/ws/roles/client which "$tool") ||
        error "mise has no $tool for /opt/ws"
    cmp -s "/usr/local/bin/$tool" "$from" || error "/usr/local/bin/$tool is not mise's $from"
    owner=$(stat -f '%Su:%Sg %Lp' "/usr/local/bin/$tool")
    [ "$owner" = "root:wheel 755" ] || error "/usr/local/bin/$tool is $owner, not root:wheel 755"
done
for want in 'fnox 1.50.0' 'age-plugin-yubikey 0.5.1'; do
    got=$("/usr/local/bin/${want% *}" --version) || true
    [ "$got" = "$want" ] || error "${want% *} --version prints '$got', not '$want'"
done

# A repository of one value, encrypted to the Nano's recipient of the plugin's trial and to a new
# SSH key. fnox reads fnox.toml from the directory it runs in, and its key from FNOX_CONFIG_DIR
# where no FNOX_AGE_KEY_FILE names one: both stay in the case's own directory.
dir=$(mktemp -d "${RUNNER_TEMP:-/tmp}/secrets.XXXXXXXX")
trap 'rm -rf "$dir"' EXIT
ssh-keygen -q -t ed25519 -N '' -C ws-ci -f "$dir/host"
cat >"$dir/fnox.toml" <<EOF
root = true
if_missing = "error"

[providers.age]
type = "age"

[providers.age.recipients]
nano = "age1yubikey1qwla8v7cu3mx6mp79asgrh5ad2h52flwln7c66ydcyy50lg5uh0gxh4kmaz"
ws-ci = "$(cut -d ' ' -f 1,2 "$dir/host.pub")"
EOF
(
    cd "$dir" &&
        export FNOX_CONFIG_DIR="$dir/config" PATH="/usr/local/bin:$PATH" &&
        unset FNOX_AGE_KEY &&
        printf v | fnox set A --provider age &&
        [ "$(FNOX_AGE_KEY_FILE="$dir/host" fnox get A)" = v ]
) || error "fnox cannot encrypt a value to the Nano's recipient and the SSH key, and read it back by the key"
exit "$fail"
