# shellcheck shell=bash
# The cases of the client job that run mise bootstrap source this file, and never run it. It sets up
# the Mac as the runbook does, by ws macos setup on the tree in /opt/ws as it stands. Setup installs
# mise by its SHA-256, writes the miserc and the trust, links the plugin and runs the bootstrap.
# ws is the build that .tools/install-build put in /usr/local/bin, which the job's step runs.

# client_trace holds the sudo that counts its calls.
client_trace=$RUNNER_TEMP/ws-trace

# client_tree puts this commit's tree in /opt/ws, root's and writable by root alone, as setup's guard
# asks. setup refuses a directory above the tree or the plugin that anyone but root or the user may
# change, as mise ignores a system plugin below one. So /opt and /usr/local/share are root's and
# 755, as on a Mac with Homebrew in /opt/homebrew, where the runner's image may differ. It drops the
# role apps, whose 14 casks setup's second run would download in each job.
client_tree() {
    sudo mkdir -p /opt /usr/local/share
    sudo chown root:wheel /opt /usr/local/share
    sudo chmod 755 /opt /usr/local/share
    sudo rm -rf /opt/ws
    sudo mkdir -p /opt/ws
    git archive HEAD | sudo tar -x -C /opt/ws
    sudo rm -rf /opt/ws/macos/roles/apps
    sudo chown -R root:wheel /opt/ws
    sudo chmod -R go-w /opt/ws
}

# client_trace_sudo writes the sudo that records each call, once.
client_trace_sudo() {
    if [ ! -x "$client_trace/sudo" ]; then
        mkdir -p "$client_trace"
        printf '#!/bin/sh\nprintf "sudo %%s\\n" "$*" | cut -c1-160 >>"%s/calls"\nexec /usr/bin/sudo "$@"\n' \
            "$client_trace" >"$client_trace/sudo"
        chmod 755 "$client_trace/sudo"
    fi
}

# client_setup ARG... runs /usr/local/bin/ws macos setup --in-place ARG... as the user, with no
# terminal. A sudo first on PATH writes a line for each call to $client_trace/calls, which client_mark
# empties and client_calls counts.
client_setup() {
    client_trace_sudo
    (cd / && PATH="$client_trace:$PATH" /usr/local/bin/ws macos setup --in-place "$@" </dev/null)
}

# client_bootstrap [plan ARG...] runs setup, which installs mise and runs mise bootstrap. The plan with
# --detailed-exitcode runs setup --plan, which exits 0 and 2 as mise's does. Any other plan runs
# mise's own, for a case that reads what it prints, as the bootstrap's first change is not done.
client_bootstrap() {
    case $* in
    '') client_setup ;;
    'plan --detailed-exitcode') client_setup --plan ;;
    *) client_tree_mise bootstrap "$@" ;;
    esac
}

# client_tree_mise ARG... runs mise -C /opt/ws/macos ARG... as the user, with no terminal, as the
# runbook runs it, with client_setup's sudo.
client_tree_mise() {
    client_trace_sudo
    (cd / && PATH="$client_trace:$PATH" mise -C /opt/ws/macos "$@" </dev/null)
}

# client_mark empties the list of sudo's calls.
client_mark() {
    mkdir -p "$client_trace"
    : >"$client_trace/calls"
}

# client_calls prints the number of sudo's calls since client_mark, then the calls themselves on stderr.
client_calls() {
    sort "$client_trace/calls" | uniq -c | sort -rn | head -12 >&2
    wc -l <"$client_trace/calls" | tr -d ' '
}
