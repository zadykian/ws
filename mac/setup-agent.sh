# shellcheck shell=sh
# The step of mac/setup.sh that installs ws's ssh agent: copies of ws-agent and its PIN's askpass in
# ~/.config/ws, and the LaunchAgent that runs ws-agent, which launchd loads at the next login.
# setup.sh sources it after lib.sh and setup-lib.sh, whose functions it calls.

# The agent's launchd label, and its plist.
agent_label=com.github.zadykian.ws-agent
agent_plist=$HOME/Library/LaunchAgents/$agent_label.plist

# agent copies ws-agent and ws-agent-pin to ~/.config/ws, and the plist to ~/Library/LaunchAgents,
# where they differ. launchd runs the copies, which keep the agent off a clone that moves or changes
# branch.
# shellcheck disable=SC2154 # ws_tmp, lib.sh's, sets ws_tmpfile, and dir is setup.sh's
agent() {
    [ -d "$HOME/.config/ws" ] || ws_dir "$HOME/.config/ws" 700
    for each in ws-agent ws-agent-pin; do
        ws_put "$HOME/.config/ws/$each" 755 <"$dir/$each"
    done
    [ -d "${agent_plist%/*}" ] || ws_dir "${agent_plist%/*}" 755
    ws_tmp
    agent_new=$ws_tmpfile
    agent_plist_text >"$agent_new"
    ws_put "$agent_plist" 644 <"$agent_new"
    rm -f "$agent_new"
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
