# shellcheck shell=bash
# The cases of the mac job that run mise bootstrap source this file, and never run it. It sets up
# the Mac as the runbook does, by ws client setup on the tree in /opt/ws as it stands. Setup installs
# mise by its SHA-256, writes the miserc and the trust, links the plugin and runs the bootstrap.
# ws is the build that .tools/install-build put in /usr/local/bin, which the job's step runs.

# mac_trace holds the sudo that counts its calls.
mac_trace=$RUNNER_TEMP/ws-trace

# mac_tree puts this commit's tree in /opt/ws, root's and writable by root alone, as setup's guard
# asks. setup refuses a directory above the tree or the plugin that anyone but root or the user may
# change, as mise ignores a system plugin below one. So /opt and /usr/local/share are root's and
# 755, as on a Mac with Homebrew in /opt/homebrew, where the runner's image may differ.
mac_tree() {
    sudo mkdir -p /opt /usr/local/share
    sudo chown root:wheel /opt /usr/local/share
    sudo chmod 755 /opt /usr/local/share
    sudo rm -rf /opt/ws
    sudo mkdir -p /opt/ws
    git archive HEAD | sudo tar -x -C /opt/ws
    sudo chown -R root:wheel /opt/ws
    sudo chmod -R go-w /opt/ws
}

# mac_trace_sudo writes the sudo that records each call, once.
mac_trace_sudo() {
    if [ ! -x "$mac_trace/sudo" ]; then
        mkdir -p "$mac_trace"
        printf '#!/bin/sh\nprintf "sudo %%s\\n" "$*" | cut -c1-160 >>"%s/calls"\nexec /usr/bin/sudo "$@"\n' \
            "$mac_trace" >"$mac_trace/sudo"
        chmod 755 "$mac_trace/sudo"
    fi
}

# mac_setup ARG... runs /usr/local/bin/ws client setup --in-place ARG... as the user, with no
# terminal. A sudo first on PATH writes a line for each call to $mac_trace/calls, which mac_mark
# empties and mac_calls counts.
mac_setup() {
    mac_trace_sudo
    (cd / && PATH="$mac_trace:$PATH" /usr/local/bin/ws client setup --in-place "$@" </dev/null)
}

# mac_bootstrap [plan ARG...] runs setup, which installs mise and runs mise bootstrap. The plan with
# --detailed-exitcode runs setup --plan, which exits 0 and 2 as mise's does. Any other plan runs
# mise's own, for a case that reads what it prints, as the bootstrap's first change is not done.
mac_bootstrap() {
    case $* in
    '') mac_setup ;;
    'plan --detailed-exitcode') mac_setup --plan ;;
    *) mac_tree_mise bootstrap "$@" ;;
    esac
}

# mac_tree_mise ARG... runs mise -C /opt/ws/roles/client ARG... as the user, with no terminal, as the
# runbook runs it, with mac_setup's sudo.
mac_tree_mise() {
    mac_trace_sudo
    (cd / && PATH="$mac_trace:$PATH" mise -C /opt/ws/roles/client "$@" </dev/null)
}

# mac_mark empties the list of sudo's calls.
mac_mark() {
    mkdir -p "$mac_trace"
    : >"$mac_trace/calls"
}

# mac_calls prints the number of sudo's calls since mac_mark, then the calls themselves on stderr.
mac_calls() {
    sort "$mac_trace/calls" | uniq -c | sort -rn | head -12 >&2
    wc -l <"$mac_trace/calls" | tr -d ' '
}
