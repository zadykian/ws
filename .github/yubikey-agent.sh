#!/bin/sh
# The job's one ws's agent, at ~/.ssh/ws-agent.sock, for the mac and wizard jobs, which
# yubikey-standin.sh runs. Where launchd has the gui domain gui/UID, it is launchd's ws-agent, which
# setup.sh loads there. Else it is a run of mac/ws-agent of the stand-in's own, as launchd loads a
# LaunchAgent at a login alone, and setup.sh then loads none:
#
#   sh .github/yubikey-agent.sh start     says which, and starts the stand-in's run where it is
#   sh .github/yubikey-agent.sh restart   starts the agent again, once a YubiKey's token changed
#
# Each waits up to 20 s for the agent to list the keys of the YubiKeys in, so that the case that
# follows finds it settled: an agent that runs keeps the keys it loaded, as on a Mac, where a
# replug or setup.sh loads them again. The stand-in's run takes the stand-in's ioreg, logs to
# $RUNNER_TEMP/yubikey/agent.log, and keeps its pid in agent.pid.

set -eu
S=${RUNNER_TEMP:?the stand-in YubiKey lives in RUNNER_TEMP}/yubikey
here=$(cd "$(dirname "$0")" && pwd -P)
# shellcheck source=mac/setup-yubikey.sh
. "$here/../mac/setup-yubikey.sh"
domain=gui/$(id -u)
job=$domain/com.github.zadykian.ws-agent

# own starts the stand-in's run of ws-agent.
own() {
    PATH=$S/bin:$PATH sh "$here/../mac/ws-agent" >>"$S/agent.log" 2>&1 </dev/null &
    echo "$!" >"$S/agent.pid"
}

# settled succeeds where the agent answers, and lists the keys that the YubiKeys in hold, each by
# its type and base64, as setup-yubikey.sh's yubikey_openssh makes them of their public keys. A
# count alone could pass on the old agent's keys, where kickstart -k returns before it ends.
settled() {
    want=$(for der in "$S"/*/[89]?/pub.der; do
        if [ -f "$der" ] && [ -f "${der%/*/pub.der}/in" ]; then
            yubikey_openssh "$der"
        fi
    done | sort)
    have=
    if listed=$(SSH_AUTH_SOCK=$HOME/.ssh/ws-agent.sock /usr/bin/ssh-add -L 2>/dev/null); then
        have=$(printf '%s\n' "$listed" | awk '{ print $1, $2 }' | sort)
    elif [ $? != 1 ]; then
        return 1
    fi
    [ "$have" = "$want" ]
}

# settle waits up to 20 s for settled, and stops the stand-in where the agent never settles.
settle() {
    tries=0
    until settled; do
        if [ "$tries" -ge 20 ]; then
            echo "::error::ws's agent lists other keys than the stand-in's YubiKeys in hold," \
                "after 20 s: $(SSH_AUTH_SOCK=$HOME/.ssh/ws-agent.sock /usr/bin/ssh-add -l 2>&1)"
            exit 1
        fi
        tries=$((tries + 1))
        sleep 1
    done
}

case ${1-} in
start)
    if launchctl print "$domain" >/dev/null 2>&1; then
        echo "::notice::ws's agent in this job is launchd's ws-agent, in $domain, once setup.sh" \
            "loads it"
        exit 0
    fi
    echo "::notice::launchd has no $domain here, so ws's agent in this job is a run of" \
        "mac/ws-agent of the stand-in's own: setup.sh loads no LaunchAgent"
    own
    ;;
restart)
    if launchctl print "$job" >/dev/null 2>&1; then
        launchctl kickstart -k "$job"
    elif [ -f "$S/agent.pid" ]; then
        pid=$(cat "$S/agent.pid")
        kill "$pid" 2>/dev/null || true
        tries=0
        while kill -0 "$pid" 2>/dev/null && [ "$tries" -lt 50 ]; do
            tries=$((tries + 1))
            sleep 0.1
        done
        own
    else
        exit 0
    fi
    ;;
*)
    echo "usage: sh .github/yubikey-agent.sh start | restart" >&2
    exit 2
    ;;
esac
settle
