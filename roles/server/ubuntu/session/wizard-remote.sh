#!/bin/sh
# The wizard's remote script on ws, which the Mac's wizard sends over ssh, to run as root there,
# through sudo where ssh logs in as another user. So the script is the Mac's clone's, whatever ws's
# clone holds, or where ws has none:
#
#   ssh -o HostName=ADDRESS -o Port=PORT ws sh -s -- OP [ARG...] <roles/server/ubuntu/session/wizard-remote.sh
#
# - status prints what the wizard reads of ws, a line each, as NAME VALUE, with - for a value that
#   is missing. user is the ws_user that host_vars names. run is the ws version and status of the
#   last full run, and off where host_vars turned the outbound table off as the run began, else on.
#   Then come the SHA-256 of that run's secrets file and of host_vars, each or -. ws is the version
#   that ws prints. last is the unit and status of the last run. active is the run's unit that runs
#   now. mac.pub is the Mac's key that wg0's peer takes, and wg0 is wg0's own key.
#   signing is the type and base64 of ws's public signing key, or none, and hostname is ws's. claude
#   is ok, none or login. fans is refused where Secure Boot refuses nct6687, the fans' driver, else
#   ok. peer is ok where wg0 has mac.pub's key as a peer. outbound
#   is off where host_vars, /etc/ws/host_vars.yml, turns the security role's outbound table off, else
#   on. cable is the port of the Mac's cable that host_vars and the network role's netplan file name.
#   secrets is the SHA-256 of ws's secrets file, as a run would stamp it, or -, and vars that of
#   host_vars.
# - state prints ws server ubuntu state --json, which changes nothing, on stdout alone. It has the
# HOME and
#   the locale of a run's unit, so that it reads what a run would.
# - run [ARG...] runs ws server ubuntu apply --json ARGs in a unit of its own, ws-wizard-STAMP,
# which goes
#   on where ssh drops. Or it follows such a unit that runs already. A unit of other ARGs would
#   leave these unrun, so run refuses it, unless that unit is a full one. It prints the unit's log
#   until it ends, and exits with the run's status. A run with no ARG is a full one. Where ws's
#   secrets file and its identity's credential are both there, the unit loads the credential and
#   runs with --secrets.
# - install TAG URL installs ws's release TAG from the releases at URL. It runs that release's
#   install.sh, which checks the binary against the release's ws.sha256 file. TAG is vX.Y.Z, and
#   URL starts with https://, or with http://127.0.0.1: for CI's stand-in, as the wizard checks them.
# - outbound names security_outbound: false in ws's host_vars, which keeps the outbound table off
#   for a cloud VM. It runs before the run, as ws server ubuntu apply reads its values from that
# file.
#
# Everything is in functions that the last line calls, so that a script cut short runs nothing.

set -eu

# The wizard's stamps, ws's binary, the values and secrets file that ws server ubuntu apply reads,
# and their identity's credential. Then the Mac's key for wg0, the fans' driver as loaded, and the
# cable's netplan file. Each is a line NAME=PATH, which CI's stand-in roots in a directory of its
# own.
state=/var/lib/ws-wizard
ws=/usr/local/bin/ws
vars=/etc/ws/host_vars.yml
secrets=/etc/ws/secrets.age
credential=/etc/credstore.encrypted/ws-secrets
peer=/etc/wireguard/mac.pub
module=/sys/module/nct6687
netplan=/etc/netplan/60-direct-link.yaml

# The user whom ssh logs in as, whose claude and git config the status reads: sudo's SUDO_USER, or
# root at a root login. ws server ubuntu wizard names WS_USER as SUDO_USER. The git config names
# ws's signing key.
user=${SUDO_USER:-$(id -un)}
home=$(getent passwd "$user" | cut -d : -f 6)
claude=$home/.local/bin/claude
gitconfig=$home/.gitconfig

say() {
    printf 'wizard-remote.sh: %s\n' "$*" >&2
}

die() {
    say "$*"
    exit 1
}

# first prints the first line of its stdin, or - where it has none.
first() {
    awk 'NR == 1 { print; found = 1 } END { if (!found) print "-" }'
}

# stamp FILE [N] prints the N words of the stamp FILE, 2 by default, or - where it has none.
stamp() {
    awk -v n="${2:-2}" 'NF == n { print; exit }' "$1" 2>/dev/null | first
}

# cable prints the port that host_vars names as network_direct_link_interface, as link writes it,
# bare or in either quotes: the last such line. It prints - for none, and where the network role's
# netplan file lacks it, as after a roles' run that failed, so that the Cable stage runs them again.
cable() {
    port=$(awk -v q="'" '/^network_direct_link_interface[ \t]*:/ {
        v = $0
        sub(/^[^:]*:[ \t]*/, "", v)
        sub(/[ \t]+#.*$/, "", v)
        sub(/[ \t\r]+$/, "", v)
        if (v ~ /^".*"$/ || v ~ ("^" q ".*" q "$")) v = substr(v, 2, length(v) - 2)
        port = (v ~ /^[A-Za-z0-9_.-]+$/) ? v : ""
    } END { print port }' "$vars" 2>/dev/null) || port=
    [ -n "$port" ] && grep -qF "\"$port\":" "$netplan" 2>/dev/null || port=-
    echo "$port"
}

active() {
    systemctl list-units --type=service --state=active,activating --no-legend --plain \
        'ws-wizard-*' 2>/dev/null | awk '{ sub(/\.service$/, "", $1); print $1; exit }'
}

# as_login runs a command as that user. The user can write those files, so root never runs them,
# nor reads through them: that would hand root to any process of the user's.
as_login() {
    runuser -u "$user" -- "$@"
}

# signing_key prints the type and base64 of the public key that the user's git config signs with.
# That is the .pub that user.signingkey names, or none where that config names no key file.
signing_key() {
    if pub=$(as_login git config -f "$gitconfig" user.signingkey) && as_login test -f "$pub"; then
        as_login cut -d ' ' -f 1-2 "$pub"
    else
        echo none
    fi
}

# status prints what the wizard reads of ws, as the header says.
status() {
    echo "run $(stamp "$state/run" 5)"
    echo "last $(stamp "$state/last")"
    echo "active $(active | first)"
    echo "ws $("$ws" version 2>/dev/null | first)"
    echo "user $(sed -n "s/^ws_user:[[:space:]]*[\"']\{0,1\}\([^\"' #]*\).*/\1/p" "$vars" \
        2>/dev/null | tail -n 1 | first)"
    echo "mac.pub $(head -n 1 "$peer" 2>/dev/null | first)"
    echo "wg0 $(wg show wg0 public-key 2>/dev/null | first)"
    echo "signing $(signing_key | first)"
    echo "hostname $(hostname | first)"
    if ! as_login test -x "$claude"; then
        echo 'claude none'
    elif as_login "$claude" auth status >/dev/null 2>&1; then
        echo 'claude ok'
    else
        echo 'claude login'
    fi
    if modinfo nct6687 >/dev/null 2>&1 && [ ! -d "$module" ] &&
        mokutil --sb-state 2>/dev/null | grep -q 'SecureBoot enabled'; then
        echo 'fans refused'
    else
        echo 'fans ok'
    fi
    echo "cable $(cable)"
    mac=$(head -n 1 "$peer" 2>/dev/null) || mac=
    echo "peer $(wg show wg0 peers 2>/dev/null |
        awk -v mac="$mac" 'mac != "" && $0 == mac { print "ok"; exit }' | first)"
    echo "outbound $(off | sh -s -- check "$vars" && echo off || echo on)"
    echo "secrets $( ([ -f "$credential" ] && (sha256sum || shasum -a 256) <"$secrets") 2>/dev/null |
        cut -d ' ' -f 1 | first)"
    echo "vars $( ([ -f "$vars" ] && (sha256sum || shasum -a 256) <"$vars") 2>/dev/null |
        cut -d ' ' -f 1 | first)"
}

# job prints the script of a run's unit, which systemd-run runs with UNIT VARS WS STATE SECRETS
# [ARG...]. It runs WS server ubuntu apply --json ARGs, whose event lines reach the wizard through
# the unit's log. Its last steps write STATE/last as UNIT and the status, each by a rename. A full
# run, which has no ARG, writes STATE/run too. It holds the version that WS printed before the run,
# or - where WS prints none, and the status. Then off where host_vars turned the outbound table off
# as the run began, else on. Last come the SHA-256 of the secrets file, or -, and that of host_vars
# as the run began, or -. It removes it as it starts. A unit with the identity as its credential
# adds --secrets and the file.
job() {
    cat <<'EOF'
unit=$1 vars=$2 ws=$3 state=$4 secrets=$5 full=0 table=on sum=- values=-
shift 5
[ $# != 0 ] || full=1
if [ -n "${CREDENTIALS_DIRECTORY-}" ]; then
    sum=$( (sha256sum || shasum -a 256) <"$secrets" | cut -d ' ' -f 1) && [ -n "$sum" ] || exit 1
    set -- "$@" --secrets "$secrets"
fi
if [ -f "$vars" ]; then
    values=$( (sha256sum 2>/dev/null || shasum -a 256) <"$vars" | cut -d ' ' -f 1) &&
        [ -n "$values" ] || exit 1
fi
sh "$state/off" check "$vars" && table=off
[ "$full" = 0 ] || rm -f "$state/run"
version=$("$ws" version 2>/dev/null | head -n 1) && [ -n "$version" ] || version=-
"$ws" server ubuntu apply --json "$@"
status=$?
if [ "$full" = 1 ]; then
    echo "$version $status $table $sum $values" >"$state/run.new" && mv -f "$state/run.new" "$state/run"
fi
echo "$unit $status" >"$state/last.new" && mv -f "$state/last.new" "$state/last"
exit "$status"
EOF
}

# off prints the script of host_vars VARS's security_outbound line, which runs as OP VARS. check
# succeeds where the last such line is false, no or off, in any case, as YAML reads it. A comment,
# blanks or a carriage return may follow. write names security_outbound: false otherwise, by a
# rename that keeps the file's other lines and mode, and not through a link. It makes the directory
# 0755 and a new file 0600, as the file may hold secrets.
off() {
    cat <<'EOF'
vars=$2
awk '/^security_outbound:/ { line = $0 } END { print line }' "$vars" 2>/dev/null |
    grep -Eiqx 'security_outbound:[[:space:]]*(false|no|off)([[:space:]]*|[[:space:]]+#.*)' && exit 0
[ "$1" = write ] || exit 1
if [ -L "$vars" ]; then
    echo "wizard-remote.sh: $vars is a link to $(readlink "$vars"): name the line in its file by hand" >&2
    exit 1
fi
[ -d "${vars%/*}" ] || { mkdir -p "${vars%/*}" && chmod 755 "${vars%/*}"; }
umask 077
[ ! -f "$vars" ] || cp -p "$vars" "$vars.new"
{
    [ -s "$vars" ] || echo ---
    cat "$vars" 2>/dev/null | awk '
        /^security_outbound:/ { if (!done++) print "security_outbound: false"; next }
        { print }
        END { if (!done) print "security_outbound: false" }'
} >"$vars.new" && mv -f "$vars.new" "$vars" && echo "wizard-remote.sh: wrote $vars" >&2
EOF
}

# run starts a run's unit, or follows the one that runs, as the header says. The unit takes root's
# HOME, which systemd gives a unit only with a user, and a UTF-8 locale.
run() {
    unit=$(active)
    if [ -n "$unit" ]; then
        other=$(cat "$state/args" 2>/dev/null) || other=
        [ -z "$other" ] || [ "$other" = "$*" ] ||
            die "$unit runs other steps on ws: journalctl -u $unit shows its log; run the wizard again once it ends"
        say "following $unit, which runs on ws already"
    else
        mkdir -p "$state"
        job >"$state/job.new" && mv -f "$state/job.new" "$state/job"
        off >"$state/off.new" && mv -f "$state/off.new" "$state/off"
        printf '%s\n' "$*" >"$state/args"
        unit=ws-wizard-$(date +%Y%m%d%H%M%S)-$$
        creds=
        [ ! -f "$secrets" ] || [ ! -f "$credential" ] || creds="-p LoadCredentialEncrypted=ws-secrets -p Type=exec"
        # shellcheck disable=SC2086 # the words are options, and no path holds a blank
        systemd-run $creds --quiet --unit="$unit" --collect --expand-environment=no \
            --description="ws server ubuntu apply, for the Mac's wizard" --setenv=LC_ALL=C.UTF-8 \
            --setenv=HOME=/root /bin/sh "$state/job" "$unit" "$vars" "$ws" "$state" "$secrets" "$@" ||
            die "systemd-run cannot start $unit"
        say "started $unit, which goes on where ssh drops: journalctl -u $unit shows its log"
    fi
    follow "$unit"
}

# follow UNIT prints UNIT's log every 2 s while it runs, and once more a second after it ends. It
# exits with the status that UNIT's job wrote in last, or 1 where the job ended before it wrote one.
follow() {
    cursor=$(mktemp -d)
    trap 'rm -rf "$cursor"' EXIT
    trap 'exit 1' HUP INT TERM
    while running "$1"; do
        journal "$1"
        sleep 2
    done
    sleep 1
    journal "$1"
    last=$(stamp "$state/last")
    [ "${last% *}" = "$1" ] ||
        die "$1 ended before its run wrote its status, as where it was stopped: run it again"
    [ "${last#* }" = 0 ] || say "$1's run exited ${last#* }: journalctl -u $1 shows its log"
    exit "${last#* }"
}

# install TAG URL installs ws's release TAG from the releases at URL, as the header says. It
# downloads that release's install.sh, which runs as root here, so both arguments are checked first.
install() {
    printf '%s\n' "$1" | grep -Eqx 'v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)' ||
        die "'$1' is not a release, vX.Y.Z"
    printf '%s\n' "$2" | grep -Eqx '(https://|http://127\.0\.0\.1:)[A-Za-z0-9._~:/-]+' ||
        die "'$2' is not an address of releases: https://, or http://127.0.0.1:"
    mkdir -p "$state" || die "cannot make $state"
    curl -fsSL --retry 3 --proto-redir =https -o "$state/install.sh" "$2/download/$1/install.sh" ||
        die "cannot download $2/download/$1/install.sh"
    WS_VERSION=$1 WS_RELEASES_URL=$2 sh "$state/install.sh"
}

running() {
    systemctl show -p ActiveState --value "$1" | grep -qxE "active|activating|deactivating"
}

# journal prints the lines of UNIT's log that follow those it printed before, as its cursor marks.
journal() {
    journalctl -q -u "$1" -o cat --no-pager --cursor-file="$cursor/at" || true
}

main() {
    [ "$(id -u)" = 0 ] || die "it runs as root on ws, through sudo or a root login"
    case ${1-} in
    status) status ;;
    state) HOME=/root LC_ALL=C.UTF-8 "$ws" server ubuntu state --json ;;
    run) shift && run "$@" ;;
    outbound) off | sh -s -- write "$vars" ;;
    install) install "${2-}" "${3-}" ;;
    *) die "unknown op '${1-}'; usage: wizard-remote.sh status | state | run [ARG...] |" \
        "install TAG URL | outbound" ;;
    esac
}

# Nothing it runs reads stdin, which under sh -s is the rest of this script. The braces make a
# script cut short a syntax error, rather than main run.
{ main "$@" </dev/null; }
