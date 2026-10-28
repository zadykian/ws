# shellcheck shell=sh
# The step of mac/setup.sh that installs ws's ssh agent: copies of ws-agent and its askpasses in
# ~/.config/ws, and the LaunchAgent that runs ws-agent, loaded in the gui domain of the user's
# login. setup.sh sources it after lib.sh, setup-lib.sh and setup-yubikey.sh, whose functions and
# yubikey_dir it takes, and runs agent after the YubiKey's step, as the wizard's Mac key stage does.

# The agent's launchd label, and its plist.
agent_label=com.github.zadykian.ws-agent
agent_plist=$HOME/Library/LaunchAgents/$agent_label.plist

# agent copies ws-agent and its askpasses, ws-agent-pin and ws-agent-confirm, to ~/.config/ws, and
# the plist to ~/Library/LaunchAgents, where they differ, then has launchd run the agent. launchd
# runs the copies, which keep the agent off a clone that moves or changes branch. The askpasses go
# first, so that an agent that starts in between finds them: one with no ws-agent-confirm refuses
# each signature.
# shellcheck disable=SC2154 # ws_tmp, lib.sh's, sets ws_tmpfile, and dir is setup.sh's
agent() {
    [ -d "$HOME/.config/ws" ] || ws_dir "$HOME/.config/ws" 700
    ws_changed=0
    for each in ws-agent-pin ws-agent-confirm ws-agent; do
        ws_put "$HOME/.config/ws/$each" 755 <"$dir/$each"
    done
    agent_copied=$ws_changed
    [ -d "${agent_plist%/*}" ] || ws_dir "${agent_plist%/*}" 755
    ws_tmp
    agent_new=$ws_tmpfile
    agent_plist_text >"$agent_new"
    ws_changed=0
    ws_put "$agent_plist" 644 <"$agent_new"
    agent_plist_changed=$ws_changed
    rm -f "$agent_new"
    agent_launchd
}

# agent_plist_text prints mac/'s plist with the home's path in place of each ~/ that opens a
# string, as launchd expands no ~.
agent_plist_text() {
    while IFS= read -r agent_line; do
        case $agent_line in
        *'<string>~/'*)
            agent_line="${agent_line%%'<string>~/'*}<string>$HOME/${agent_line#*'<string>~/'}"
            ;;
        esac
        printf '%s\n' "$agent_line"
    done <"$dir/$agent_label.plist"
}

# agent_launchd has launchd run the agent in gui/UID, the domain of the user's login at the
# screen, as setup.sh's launchd does for the daemon: bootstrap where it isn't loaded, bootout and
# bootstrap where the plist changed, and kickstart -k where a copy changed, or the agent lacks a
# key, as one made since it loaded the YubiKey's. Then it waits for the keys. With no such login,
# as over ssh alone, the domain is missing, and launchd loads the plist at the next.
agent_launchd() {
    agent_domain=gui/$(id -u)
    launchctl print "$agent_domain" >/dev/null 2>&1 || return 0
    if ! launchctl print "$agent_domain/$agent_label" >/dev/null 2>&1; then
        agent_do bootstrap "$agent_domain" "$agent_plist"
    elif [ "$agent_plist_changed" = 1 ]; then
        agent_do bootout "$agent_domain/$agent_label"
        agent_do bootstrap "$agent_domain" "$agent_plist"
    elif [ "$agent_copied" = 1 ] || ! agent_holds; then
        agent_do kickstart -k "$agent_domain/$agent_label"
    else
        return 0
    fi
    agent_wait
}

# agent_holds succeeds where ws's agent lists the key of each .pub of yubikey_dir that
# WS_SSH_KEY and WS_GIT_KEY name, by its type and base64.
# shellcheck disable=SC2154 # yubikey_dir is setup-yubikey.sh's, and ws_agent_socket setup-conf.sh's
agent_holds() {
    agent_listed=$(SSH_AUTH_SOCK=$ws_agent_socket /usr/bin/ssh-add -L 2>/dev/null) || agent_listed=
    for agent_name in "$WS_SSH_KEY" "$WS_GIT_KEY"; do
        [ -f "$yubikey_dir/$agent_name.pub" ] || continue
        printf '%s\n' "$agent_listed" | awk -v key="$(awk 'NF { print $1, $2; exit }' \
            "$yubikey_dir/$agent_name.pub")" '$1 " " $2 == key { found = 1 } END { exit !found }' ||
            return 1
    done
}

# agent_wait waits up to 10 s for the agent to hold the keys, as it loads them as it starts, and
# warns where it lacks one still: its log says why.
agent_wait() {
    ws_checking && return 0
    agent_tries=0
    until agent_holds; do
        if [ "$agent_tries" -ge 10 ]; then
            ws_say "warning: ws's agent holds no key of $yubikey_dir/$agent_name.pub after 10 s:" \
                "its log, ~/Library/Logs/ws-agent.log, says why"
            return 0
        fi
        agent_tries=$((agent_tries + 1))
        sleep 1
    done
}

# agent_do ARGS... runs launchctl ARGS, as setup.sh's launchd_do runs sudo launchctl for the
# daemon. A bootstrap right after a bootout may find the job still on its way out, so it tries for
# 10 s.
agent_do() {
    if ws_checking; then
        ws_say "would run launchctl $*"
        return 0
    fi
    agent_tries=1
    until agent_err=$(launchctl "$@" 2>&1); do
        if [ "$1" != bootstrap ] || [ "$agent_tries" -ge 10 ]; then
            ws_die "launchctl $* failed: $agent_err; is ws-agent on under System Settings ›" \
                "General › Login Items & Extensions, Allow in the Background?"
        fi
        agent_tries=$((agent_tries + 1))
        sleep 1
    done
    ws_say "ran launchctl $*"
}
