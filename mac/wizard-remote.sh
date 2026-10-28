#!/bin/sh
# The wizard's remote script on ws, which the Mac's wizard sends to root there over ssh. So it is
# the Mac's clone's, whatever ws's clone holds, or where ws has none:
#
#   ssh ws-ssh sh -s -- OP [ARG...] <mac/wizard-remote.sh
#
# - status prints what the wizard reads of ws, a line each, as NAME VALUE, with - for a value that
#   is missing. run is the commit and status of the last full run, and off where its -e turned the
#   outbound table off, else on. last is the unit and status of the last run. active is the run's
#   unit that runs now. mac.pub is the Mac's key that wg0's peer
#   takes, and wg0 is wg0's own key. gh is ok, none, login, scope or key, below. claude is ok, none
#   or login. fans is refused where Secure Boot refuses nct6687, the fans' driver, else ok. peer is
#   ok where wg0 has mac.pub's key as a peer. hook lists the repositories, as OWNER/REPO, whose
#   apply.sh has a stamp of wizard-remote-files.sh's, the files script. outbound is off where the
#   clone's host_vars turns the security role's outbound table off, else on.
#   cable is the port of the Mac's cable that host_vars and the network role's netplan file name.
# - run [--update] [ARG...] runs bootstrap.sh ARGs in a unit of its own, ws-wizard-STAMP, which goes
#   on where ssh drops, or follows such a unit that runs already. It prints the unit's log until it
#   ends, and exits with the run's status. --update first pulls a clean clone on main. A run whose
#   ARGs are -e pairs alone is a full one.
# - outbound names security_outbound: false in ws's host_vars, where a full run's unit with -e
#   security_outbound=false didn't, which keeps the outbound table off for a cloud VM.
# - peer KEY writes KEY, the Mac's WireGuard key, to mac.pub where that holds another.
#
# Everything is in functions that the last line calls, so that a script cut short runs nothing.

set -eu

# The wizard's stamps, ws's clone, the Mac's key for wg0, root's claude, root's git config, which
# names ws's signing key, the fans' driver as the kernel loaded it, and the network role's netplan
# file of the cable. Each is a line NAME=PATH, which CI's stand-in for ws roots in a directory of
# its own.
state=/var/lib/ws-wizard
dir=/root/repository/ws
peer=/etc/wireguard/mac.pub
claude=/root/.local/bin/claude
gitconfig=/root/.gitconfig
module=/sys/module/nct6687
netplan=/etc/netplan/60-direct-link.yaml

# say prints a message of the script's on stderr.
say() {
    printf 'wizard-remote.sh: %s\n' "$*" >&2
}

# die prints a message of the script's and exits with status 1.
die() {
    say "$*"
    exit 1
}

# first prints the first line of its stdin, or - where it has none.
first() {
    awk 'NR == 1 { print; found = 1 } END { if (!found) print "-" }'
}

# stamp FILE prints the words of the stamp FILE, two or more, or - where it has none.
stamp() {
    awk 'NF >= 2 { print; exit }' "$1" 2>/dev/null | first
}

# cable prints the port that host_vars in the clone names as network_direct_link_interface, as
# link writes it, bare or in either quotes: the last such line, as Ansible reads the file. It
# prints - for none, and where the network role's netplan file lacks the port, as after a roles'
# run that failed, so that the Cable stage runs them again.
cable() {
    port=$(awk -v q="'" '/^network_direct_link_interface[ \t]*:/ {
        v = $0
        sub(/^[^:]*:[ \t]*/, "", v)
        sub(/[ \t]+#.*$/, "", v)
        sub(/[ \t\r]+$/, "", v)
        if (v ~ /^".*"$/ || v ~ ("^" q ".*" q "$")) v = substr(v, 2, length(v) - 2)
        port = (v ~ /^[A-Za-z0-9_.-]+$/) ? v : ""
    } END { print port }' "$dir/host_vars/localhost.yml" 2>/dev/null) || port=
    [ -n "$port" ] && grep -qF "\"$port\":" "$netplan" 2>/dev/null || port=-
    echo "$port"
}

# active prints the run's unit that is active or activating, where there is one.
active() {
    systemctl list-units --type=service --state=active,activating --no-legend --plain \
        'ws-wizard-*' 2>/dev/null | awk '{ sub(/\.service$/, "", $1); print $1; exit }'
}

# gh_state prints none without gh, login without a login to github.com or where its list of signing
# keys fails but for a 404, and scope at that 404. key is where the list lacks the key that root's
# git config signs with, and ok where it holds it, or where that config names no key file there.
gh_state() {
    command -v gh >/dev/null || { echo none && return; }
    gh auth status -h github.com >/dev/null 2>&1 || { echo login && return; }
    pub=$(git config -f "$gitconfig" user.signingkey) && [ -f "$pub" ] || { echo ok && return; }
    if ! keys=$(gh api --paginate user/ssh_signing_keys --jq '.[].key' 2>&1); then
        case $keys in *'HTTP 404'*) echo scope ;; *) echo login ;; esac
    elif printf '%s\n' "$keys" | grep -qxF -e "$(cut -d ' ' -f 1-2 "$pub")"; then
        echo ok
    else
        echo key
    fi
}

# status prints what the wizard reads of ws, as the header says.
status() {
    echo "run $(stamp "$state/run")"
    echo "last $(stamp "$state/last")"
    echo "active $(active | first)"
    echo "mac.pub $(head -n 1 "$peer" 2>/dev/null | first)"
    echo "wg0 $(wg show wg0 public-key 2>/dev/null | first)"
    echo "gh $(gh_state)"
    if [ ! -x "$claude" ]; then
        echo 'claude none'
    elif "$claude" auth status >/dev/null 2>&1; then
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
    hook=$(cat "$state"/hook-* 2>/dev/null | awk '$3 != "" { printf " %s", $3 }')
    echo "hook${hook:- -}"
    echo "outbound $(off | sh -s -- check "$dir/host_vars/localhost.yml" && echo off || echo on)"
}

# pull pulls the clone by fast-forward, where it is on main with nothing uncommitted, untracked
# files aside. Otherwise it warns, and leaves it as it is.
pull() {
    [ -e "$dir/.git" ] || return 0
    if [ "$(git -C "$dir" symbolic-ref -q --short HEAD)" != main ] ||
        [ -n "$(git -C "$dir" status --porcelain --untracked-files=no)" ]; then
        say "warning: leaves $dir as it is, not on main or not clean: pull it by hand"
        return 0
    fi
    head=$(git -C "$dir" rev-parse --short HEAD)
    git -C "$dir" pull -q --ff-only || die "git pull --ff-only failed in $dir; mend the clone"
    [ "$(git -C "$dir" rev-parse --short HEAD)" = "$head" ] ||
        say "pulled $dir to $(git -C "$dir" rev-parse --short HEAD)"
}

# job prints the script of a run's unit, which systemd-run runs with UNIT DIR STATE [ARG...]. It
# runs bootstrap.sh ARGs from the clone in DIR, or from the README's download where DIR holds none.
# Its last steps write STATE/last as UNIT and the status, each by a rename. A full run, whose ARGs
# are -e pairs alone, writes STATE/run too: the clone's commit, the status, and off where
# -e security_outbound=false turned the outbound table off, else on. It removes it as it starts.
# Such a run that passed first names that in host_vars by STATE/off, so that no ssh comes between.
job() {
    cat <<'EOF'
unit=$1 dir=$2 state=$3 full=1 table=on pair=
shift 3
for arg; do
    case $pair:$arg in
    1:security_outbound=false) table=off pair= ;;
    1:*) pair= ;;
    :-e) pair=1 ;;
    *) full=0 ;;
    esac
done
[ -z "$pair" ] || full=0
[ "$full" = 0 ] || rm -f "$state/run"
if [ -e "$dir/.git" ]; then
    sh "$dir/bootstrap.sh" "$@"
else
    curl -fsSL --retry 3 -o "$state/bootstrap.sh" \
        https://raw.githubusercontent.com/zadykian/ws/main/bootstrap.sh &&
        sh "$state/bootstrap.sh" "$@"
fi
status=$?
if [ "$full" = 1 ]; then
    [ "$status$table" != 0off ] || sh "$state/off" write "$dir/host_vars/localhost.yml"
    sha=$(git -C "$dir" rev-parse HEAD 2>/dev/null) || sha=-
    echo "$sha $status $table" >"$state/run.new" && mv -f "$state/run.new" "$state/run"
fi
echo "$unit $status" >"$state/last.new" && mv -f "$state/last.new" "$state/last"
exit "$status"
EOF
}

# off prints the script of host_vars VARS's security_outbound line, which runs as OP VARS. check
# succeeds where the last such line is false, no or off, in any case, with a comment or not, as YAML
# reads it. write names security_outbound: false otherwise, by a rename that keeps the file's other
# lines and mode, so neither through a link nor outside a clone.
off() {
    cat <<'EOF'
vars=$2
awk '/^security_outbound:/ { line = $0 } END { print line }' "$vars" 2>/dev/null |
    grep -Eiqx 'security_outbound:[[:space:]]*(false|no|off)([[:space:]]+#.*)?' && exit 0
[ "$1" = write ] || exit 1
dir=${vars%/host_vars/*} why=
[ ! -L "$vars" ] || why="$vars is a link to $(readlink "$vars"): name the line in its file by hand"
[ -d "$dir/.git" ] || why="ws has no clone in $dir, which ws's run makes"
[ -z "$why" ] || { echo "wizard-remote.sh: $why" >&2 && exit 1; }
mkdir -p "${vars%/*}"
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
# HOME, which systemd gives a unit only with a user, and the locale that Ansible needs.
run() {
    update=0
    [ "${1-}" != --update ] || { update=1 && shift; }
    unit=$(active)
    if [ -n "$unit" ]; then
        say "following $unit, which runs on ws already"
    else
        [ "$update" = 0 ] || pull
        mkdir -p "$state"
        job >"$state/job.new"
        mv -f "$state/job.new" "$state/job"
        off >"$state/off.new" && mv -f "$state/off.new" "$state/off"
        unit=ws-wizard-$(date +%Y%m%d%H%M%S)-$$
        systemd-run --quiet --unit="$unit" --collect --expand-environment=no \
            --description="ws's playbook, for the Mac's wizard" --setenv=LC_ALL=C.UTF-8 \
            --setenv=HOME=/root /bin/sh "$state/job" "$unit" "$dir" "$state" "$@" ||
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

# peer KEY writes KEY, a WireGuard public key, to mac.pub, which wg0's peer mac takes at the
# wireguard role's next run, where mac.pub holds another. Its directory is root's alone, as
# wireguard-tools makes it.
peer() {
    printf '%s\n' "$1" | grep -qxE '[A-Za-z0-9+/]{43}=' && [ "${#1}" = 44 ] ||
        die "'$1' is not a WireGuard public key"
    [ "$(head -n 1 "$peer" 2>/dev/null)" != "$1" ] || return 0
    (umask 077 && mkdir -p "${peer%/*}") || die "cannot make ${peer%/*}"
    printf '%s\n' "$1" >"$peer.new" || die "cannot write $peer.new"
    mv -f "$peer.new" "$peer" || die "cannot write $peer"
    say "wrote $peer"
}

# running succeeds while UNIT is active, or on its way in or out.
running() {
    systemctl show -p ActiveState --value "$1" | grep -qxE "active|activating|deactivating"
}

# journal prints the lines of UNIT's log that follow those it printed before, as its cursor marks.
journal() {
    journalctl -q -u "$1" -o cat --no-pager --cursor-file="$cursor/at" || true
}

main() {
    [ "$(id -u)" = 0 ] || die "it runs as root on ws, as ssh ws-ssh logs in"
    case ${1-} in
    status) status ;;
    run) shift && run "$@" ;;
    outbound) off | sh -s -- write "$dir/host_vars/localhost.yml" ;;
    peer) peer "${2-}" ;;
    *) die "unknown op '${1-}'; usage: wizard-remote.sh status | run [--update] [ARG...] |" \
        "outbound | peer KEY" ;;
    esac
}

# Nothing it runs reads stdin, which under sh -s is the rest of this script. The braces make a
# script cut short a syntax error, rather than main run.
{ main "$@" </dev/null; }
