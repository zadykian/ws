# shellcheck shell=bash
# The cases of the mac job that run mise bootstrap source this file, and never run it. It sets up
# the Mac as the runbook does. mise goes to /usr/local/bin by its SHA-256, a miserc names the env
# mac, and the tree goes to /opt/ws, root's. ws's build stands in for the release that mise.toml pins,
# served on loopback by .tools/serve-build's serve-releases.

# mac_mise_version and mac_mise_sum pin mise and the SHA-256 of its macos-arm64 file, as the runbook
# does.
mac_mise_version=2026.11.9
mac_mise_sum=f3bb0e8cbe0217a0b52444ecc0042364563ddd1de0490313bb7e702ecab41a14

# mac_rel is .tools/serve-build's directory, and mac_trace holds the sudo that counts its calls.
mac_rel=$RUNNER_TEMP/ws-rel
mac_trace=$RUNNER_TEMP/ws-trace

# mac_mise puts the pinned mise in /usr/local/bin, once its SHA-256 holds, and /etc/mise/miserc.toml.
mac_mise() {
    curl -fsSL --retry 3 -o "$RUNNER_TEMP/mise" \
        "https://github.com/jdx/mise/releases/download/v$mac_mise_version/mise-v$mac_mise_version-macos-arm64"
    if [ "$(shasum -a 256 <"$RUNNER_TEMP/mise" | awk '{ print $1 }')" != "$mac_mise_sum" ]; then
        echo "::error::mise $mac_mise_version's SHA-256 is not the pinned one"
        return 1
    fi
    sudo mkdir -p /usr/local/bin /etc/mise
    sudo install -m 0755 "$RUNNER_TEMP/mise" /usr/local/bin/mise
    printf '# The Mac: mise loads mise.mac.toml and the Mac'"'"'s roles.\nenv = ["mac"]\n' |
        sudo tee /etc/mise/miserc.toml >/dev/null
}

# mac_tree puts this commit's tree in /opt/ws, root's and writable by root alone, and links its
# plugin into /usr/local/share/mise/plugins, as the runbook does. mise refuses a system plugin that
# anyone but root or the user may change, through any directory on its way. So /opt and
# /usr/local/share are root's and 755, as on a Mac with Homebrew in /opt/homebrew, where the
# runner's image may differ. ws's build goes beside the tree, as the pinned version, and mac_serve
# serves it.
mac_tree() {
    sh .tools/serve-build . "$mac_rel" darwin arm64
    mac_serve
    sudo mkdir -p /opt /usr/local/share/mise/plugins
    sudo chown root:wheel /opt /usr/local/share
    sudo chmod 755 /opt /usr/local/share
    sudo rm -rf /opt/ws
    sudo mkdir -p /opt/ws
    git archive HEAD | sudo tar -x -C /opt/ws
    sudo cp mise.local.toml mise.local.lock /opt/ws/
    sudo chown -R root:wheel /opt/ws
    sudo chmod -R go-w /opt/ws
    sudo ln -sfn /opt/ws/plugins/ws /usr/local/share/mise/plugins/ws
}

# mac_serve serves mac_tree's build on port 8765, by serve-releases, for a later case's bootstrap
# too. mac_server is its pid, which the caller stops.
mac_serve() {
    rm -f "$mac_rel/port"
    "$mac_rel/serve-releases" "$mac_rel" 8765 "$mac_rel/port" &
    # shellcheck disable=SC2034 # the case that sources this file stops it
    mac_server=$!
    for _ in $(seq 50); do
        [ -s "$mac_rel/port" ] && break
        sleep 0.1
    done
    if [ ! -s "$mac_rel/port" ]; then
        echo '::error::serve-releases did not start'
        return 1
    fi
}

# mac_build prints the path of ws's build that mac_tree serves.
mac_build() {
    echo "$mac_rel"/download/v*/ws-darwin-arm64
}

# mac_bootstrap ARG... runs mise -C /opt/ws/roles/client bootstrap ARG..., as mac_tree_mise does.
mac_bootstrap() {
    mac_tree_mise bootstrap "$@"
}

# mac_tree_mise ARG... runs mise -C /opt/ws/roles/client ARG... as the user, locked, with no
# terminal, as the runbook runs it. A sudo first on PATH writes a line for each call to
# $mac_trace/calls, which mac_mark empties and mac_calls counts.
mac_tree_mise() {
    if [ ! -x "$mac_trace/sudo" ]; then
        mkdir -p "$mac_trace"
        printf '#!/bin/sh\nprintf "sudo %%s\\n" "$*" | cut -c1-160 >>"%s/calls"\nexec /usr/bin/sudo "$@"\n' \
            "$mac_trace" >"$mac_trace/sudo"
        chmod 755 "$mac_trace/sudo"
    fi
    (
        cd / &&
            MISE_LOCKED=1 MISE_TRUSTED_CONFIG_PATHS=/opt/ws PATH="$mac_trace:$PATH" \
                mise -C /opt/ws/roles/client "$@" </dev/null
    )
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
