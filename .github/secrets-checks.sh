#!/bin/bash
# The rootless job's cases of the secrets task, as root on the runner that rootless-checks.sh set
# up (.github/workflows/rootless.md). A bare repository on disk stands in for GitHub's, and the
# runner's host key opens its values, as a server's does. Each refusal keeps the secret file, and
# no output of the task's holds a value.
set -euo pipefail

user=zadykian
home=/home/$user
export HOME=/root MISE_LOCKED=1 MISE_DATA_DIR=/usr/local/share/mise
check=$(cd "$(dirname "$0")" && pwd)/warnings-check.sh
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT
seed=$work/seed
file=/etc/ws-ci-secrets

fail() {
    echo "::error::$*"
    exit 1
}

must() {
    local name=$1
    shift
    "$@" || fail "$name"
    echo "ok: $name"
}

refused() {
    local name=$1
    shift
    ! "$@" || fail "$name"
    echo "ok: $name"
}

as() {
    runuser -u "$user" -- env -C "$home" HOME="$home" "$@"
}

quiet() {
    "$@" >/dev/null 2>&1
}

# The repository, which the runner's own mise files point the task at. mise.local.toml is
# .tools/serve-build's, which git ignores. mise.secrets.toml loads under the env secrets alone, as
# a role's does: two inputs, one shared and one of the profile, make one file of root's.
printf '\n[env]\nWS_SECRETS_REPO = "%s"\n' "$work/secrets.git" >>/opt/ws/mise.local.toml
cat >/opt/ws/mise.secrets.toml <<'TOML'
# CI's inputs of the secrets task, which .github/secrets-checks.sh writes: never committed.
[bootstrap.secrets]
WS_CI_SHARED = { env = "WS_CI_SHARED", allow_empty = true }
WS_CI_HOST = { env = "WS_CI_HOST", allow_empty = true }

[bootstrap.files."/etc/ws-ci-secrets"]
template = true
owner = "root"
group = "root"
mode = "0600"
content = '{{ secret(name="WS_CI_SHARED") }} {{ secret(name="WS_CI_HOST") }}'
TOML
mise trust /opt/ws/mise.local.toml
mise trust /opt/ws/mise.secrets.toml

# The seed, as the Mac's init writes it. The host key is a recipient of the shared values, and of
# its profile's, which a provider of its own encrypts. No config of root's takes part.
host=$(hostname)
fnox=$(mise -C /opt/ws/roles/server which fnox)
mkdir "$work/fnox"
fx() {
    env -C "$seed" FNOX_CONFIG_DIR="$work/fnox" FNOX_DAEMON=off "$fnox" "$@" >/dev/null
}
gs() {
    GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1 git -C "$seed" -c user.name=ci \
        -c user.email=ci@ws.invalid -c commit.gpgsign=false "$@"
}
publish() {
    gs add -A
    gs commit -q -m "$1"
    gs push -q origin main
}
# profile KEY writes the profile of the host, with KEY, an ssh .pub, its one recipient.
profile() {
    printf '[providers.host]\ntype = "age"\n\n[providers.host.recipients]\n%s = "%s"\n' \
        "$host" "$(cut -d' ' -f1,2 "$1")" >"$seed/fnox.$host.toml"
}
# value NAME PROVIDER [-P HOST] seeds NAME with a new value, which it prints, masked in the log.
values=()
value() {
    local name=$1 provider=$2 v
    shift 2
    v=$(openssl rand -hex 16)
    echo "::add-mask::$v" >&2
    values+=("$v")
    printf %s "$v" | fx "$@" set "$name" --provider "$provider"
    printf %s "$v"
}
git init -q --bare -b main "$work/secrets.git"
git init -q -b main "$seed"
gs remote add origin "$work/secrets.git"
cat >"$seed/fnox.toml" <<TOML
root = true
if_missing = "error"

[providers.age]
type = "age"

[providers.age.recipients]
$host = "$(cut -d' ' -f1,2 /etc/ssh/ssh_host_ed25519_key.pub)"
TOML
profile /etc/ssh/ssh_host_ed25519_key.pub
value WS_CI_SHARED age >"$work/shared"
value WS_CI_HOST host -P "$host" >"$work/host"
# Every other input that mise lists is declared empty, as the Mac's init declares each.
inputs=$(mise -C /opt/ws/roles/server -E ws,secrets bootstrap secrets status --json | jq -r '.[].env')
for name in $inputs; do
    grep -qE "^$name *=" "$seed/fnox.toml" "$seed/fnox.$host.toml" ||
        fx set "$name" --provider age </dev/null
done
publish "ci: seed the secrets"

# task runs the secrets task as the runbook gives it, and keeps its output in $work/out.
task() {
    "$check" -- mise -C /opt/ws/roles/server run secrets -- "$@" </dev/null 2>&1 | tee -a "$work/out"
    return "${PIPESTATUS[0]}"
}
boot() { "$check" -- mise -C /opt/ws/roles/server bootstrap "$@" </dev/null; }
holds() { [ "$(cat "$file")" = "$(cat "$work/shared") $(cat "$work/host")" ]; }

must "the first run clones the repository and runs the bootstrap" task
must "the file holds both values" holds
must "the file is root's 600" test "$(stat -c '%U %a' "$file")" = 'root 600'
must "the clone is root's 700" test "$(stat -c '%U %a' /var/lib/ws/secrets)" = 'root 700'
must "the first run through the task leaves nothing to do" task plan --detailed-exitcode
must "a plan without the env keeps the file" boot plan --detailed-exitcode
value WS_CI_SHARED age >"$work/shared"
publish "ci: change the shared value"
must "the second run fast-forwards the clone" task
must "the file holds the new value" holds
must "the second run through the task leaves nothing to do" task plan --detailed-exitcode

refused "the account reads the clone" quiet as ls -- /var/lib/ws/secrets
refused "the account reads the clone's fnox.toml" quiet as cat -- /var/lib/ws/secrets/fnox.toml
refused "the account reads the host key" quiet as cat -- /etc/ssh/ssh_host_ed25519_key
out=$(as mise -C /opt/ws/roles/server run secrets </dev/null 2>&1) && fail "the account ran the task"
must "the task refuses the account" grep -qF 'run it as root' <<<"$out"

# refusal NAME TEXT: the task fails, says TEXT, and keeps the file's bytes.
refusal() {
    local sum out
    sum=$(sha256sum "$file")
    if out=$(mise -C /opt/ws/roles/server run secrets </dev/null 2>&1); then
        fail "$1: the task ran"
    fi
    printf '%s\n' "$out" | tee -a "$work/out"
    grep -qF -- "$2" <<<"$out" || fail "$1: the task never says '$2'"
    [ "$sum" = "$(sha256sum "$file")" ] || fail "$1: the file changed"
    echo "ok: $1"
}
hint='ws client macos secrets add NAME'
touch /var/lib/ws/secrets/fnox.local.toml
refusal "a file of the clone's own is refused" 'holds changes of its own'
rm /var/lib/ws/secrets/fnox.local.toml
ssh-keygen -q -t ed25519 -N '' -C other -f "$work/other"
profile "$work/other.pub"
value WS_CI_HOST host -P "$host" >/dev/null
publish "ci: encrypt the profile to another key"
refusal "a profile that the host key can't open names the add" "$hint"
gs revert --no-edit HEAD >/dev/null
gs push -q origin main
gs mv "fnox.$host.toml" fnox.other.toml
publish "ci: rename the profile"
refusal "a repository with no profile names the add" "$hint"
gs revert --no-edit HEAD >/dev/null
gs push -q origin main
must "a run after the refusals runs" task
must "the file holds both values after the refusals" holds
must "a plan after the refusals leaves nothing to do" task plan --detailed-exitcode

refused "the task's output holds a value" grep -qF -f <(printf '%s\n' "${values[@]}") "$work/out"
