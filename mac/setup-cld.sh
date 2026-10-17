# shellcheck shell=sh
# The part of mac/setup.sh that sets up cld on a Mac with Ghostty, for cld restore tabs, which
# reopens Ghostty's tabs of cld's sessions. setup.sh sources it after setup-ssh.sh, whose header it
# uses, and calls cld_mac after ssh_files. ssh_config calls cld_ssh_lines. Each does nothing where
# Ghostty's app is missing. cld_mac warns where cld or Ghostty's files can't be set up, so that the
# run goes on to the tunnel.

# cld where its install.sh puts it, and the releases whose latest has that script.
cld_bin=$HOME/.local/bin/cld
cld_releases=https://github.com/zadykian/cld/releases
# 1 where ssh's hosts start cld's watcher, as cld_tabs_check sets it.
cld_tabs=0

# cld_ghostty succeeds where Ghostty's app is there: Ghostty.app in /Applications or
# ~/Applications. Where WS_GHOSTTY_APP is set, as on CI, the path it names stands in for both.
cld_ghostty() {
    if [ -n "${WS_GHOSTTY_APP-}" ]; then
        [ -d "$WS_GHOSTTY_APP" ]
    else
        [ -d /Applications/Ghostty.app ] || [ -d "$HOME/Applications/Ghostty.app" ]
    fi
}

# cld_tabs_check sets cld_tabs to 1 where Ghostty's app is there and the cld in ~/.local/bin has
# restore tabs, else 0. cld help restore tabs fails in a cld without it, so that ssh never runs a
# cld that would refuse restore tabs --watch at each connection.
cld_tabs_check() {
    cld_tabs=0
    if cld_ghostty && [ -x "$cld_bin" ] && "$cld_bin" help restore tabs >/dev/null 2>&1; then
        cld_tabs=1
    fi
}

# cld_ssh_lines prints the lines of Host ws ws-ssh that start cld's watcher, where cld_tabs is 1.
# ssh runs LocalCommand through the user's shell once it has connected, with %d as the Mac's home
# (ssh_config(5)). The line starts the watcher in a Ghostty pane alone, so that macOS charges its
# Apple Events to Ghostty. Elsewhere, or without cld, it does nothing and exits 0.
cld_ssh_lines() {
    [ "$cld_tabs" = 1 ] || return 0
    cat <<'EOF'
    # In a Ghostty pane, start cld's watcher, unless one runs, for cld restore tabs.
    PermitLocalCommand yes
    LocalCommand [ "$TERM_PROGRAM" != ghostty ] || [ ! -x %d/.local/bin/cld ] || %d/.local/bin/cld restore tabs --watch
EOF
}

# cld_would_tabs says, under --check, that config.d/ws would gain the lines that start cld's
# watcher where the cld to come has restore tabs. --check cannot run a cld it has not installed.
cld_would_tabs() {
    [ "$cld_tabs" = 1 ] || ws_say "would add the lines that start cld's watcher to config.d/ws," \
        "where that cld has restore tabs"
}

# cld_install installs cld into ~/.local/bin, replacing any cld there, with the install.sh of cld's
# latest release, as the claude role does on ws. install.sh checks the binary against the
# release's cld.sha256 before it moves it there. It edits no shell profile.
# shellcheck disable=SC2154 # ws_tmp, lib.sh's, sets ws_tmpfile
cld_install() {
    if ws_checking; then
        ws_say "would install cld into ~/.local/bin with the install.sh of its latest release"
        cld_would_tabs
        return 0
    fi
    ws_tmp
    cld_script=$ws_tmpfile
    if ! curl -fsSL --retry 3 --proto-redir =https "$cld_releases/latest/download/install.sh" \
        >"$cld_script"; then
        ws_say "warning: cannot download cld's install.sh from $cld_releases/latest"
    elif ! CLD_INSTALL_DIR=$HOME/.local/bin sh "$cld_script" >&2; then
        ws_say "warning: cld's install.sh failed"
    fi
    rm -f "$cld_script"
}

# cld_release VERSION succeeds where VERSION is a release's, as 0.13.0.
cld_release() {
    printf '%s\n' "$1" | grep -Eqx '[0-9]+\.[0-9]+\.[0-9]+'
}

# cld_older A B succeeds where the release A comes before the release B.
cld_older() {
    awk -v a="$1" -v b="$2" 'BEGIN {
        split(a, x, ".")
        split(b, y, ".")
        for (i = 1; i <= 3; i++)
            if (x[i] + 0 != y[i] + 0)
                exit !(x[i] + 0 < y[i] + 0)
        exit 1
    }'
}

# cld_update has cld replace itself where a newer release is out, and prints only where it did, as
# the claude role does. A release before 0.6.0 lacks cld update, so install.sh replaces it. A cld
# built from a clone, of no release's version, is left alone, as cld update refuses it.
cld_update() {
    cld_have=$("$cld_bin" --version 2>/dev/null) || cld_have=
    cld_have=${cld_have#cld }
    if ! cld_release "$cld_have"; then
        [ -n "$cld_have" ] || ws_say "warning: $cld_bin --version failed"
        return 0
    fi
    if ! "$cld_bin" help update >/dev/null 2>&1; then
        cld_install
        return 0
    fi
    if ! ws_checking; then
        cld_out=$("$cld_bin" update 2>&1) || ws_say "warning: cld update failed: $cld_out"
        case $cld_out in
        Updated*) ws_say "$cld_out" ;;
        esac
        return 0
    fi
    # --check reads the release that releases/latest leads to, as cld update does.
    if ! cld_latest=$(curl -fsSIL --retry 3 -o /dev/null -w '%{url_effective}' \
        "$cld_releases/latest"); then
        ws_say "warning: cannot read cld's latest release from $cld_releases/latest"
        return 0
    fi
    cld_latest=${cld_latest##*/v}
    cld_release "$cld_latest" && cld_older "$cld_have" "$cld_latest" || return 0
    ws_say "would run cld update: cld $cld_have, the latest $cld_latest"
    cld_would_tabs
}

# cld_ghostty_conf prints the config file that Ghostty's menu opens (src/config/edit.zig, 1.3.1
# and main). It tries config.ghostty and config in Application Support's com.mitchellh.ghostty,
# then in ~/.config/ghostty. The first that holds anything wins, else the first that exists, else
# the first. Ghostty reads all four. Its app, which launchd starts, sees no XDG_CONFIG_HOME of a
# shell's.
cld_ghostty_conf() {
    cld_support="$HOME/Library/Application Support/com.mitchellh.ghostty"
    cld_found=
    for cld_each in "$cld_support/config.ghostty" "$cld_support/config" \
        "$HOME/.config/ghostty/config.ghostty" "$HOME/.config/ghostty/config"; do
        if [ -f "$cld_each" ] && [ -s "$cld_each" ]; then
            echo "$cld_each"
            return 0
        fi
        [ -n "$cld_found" ] || [ ! -f "$cld_each" ] || cld_found=$cld_each
    done
    echo "${cld_found:-$cld_support/config.ghostty}"
}

# cld_includes CONF FILE succeeds where Ghostty's config CONF has a config-file line that names
# FILE. The line counts with or without the ? and the quotes Ghostty takes, and its path may be
# relative to CONF's directory, under ~/ or full (src/cli/args.zig's LineIterator). A path counts
# by its directory with links resolved, as Ghostty resolves a relative one (src/config/path.zig).
cld_includes() {
    [ -f "$1" ] || return 1
    cld_want=$(cd -P "${2%/*}" 2>/dev/null && pwd -P) || return 1
    cld_paths=$(awk -v dir="${1%/*}" -v home="$HOME" '
        {
            line = $0
            sub(/^[ \t]+/, "", line)
            sub(/[ \t\r]+$/, "", line)
            at = index(line, "=")
        }
        line ~ /^#/ || at == 0 { next }
        {
            key = substr(line, 1, at - 1)
            sub(/[ \t]+$/, "", key)
        }
        key != "config-file" { next }
        {
            path = substr(line, at + 1)
            sub(/^[ \t]+/, "", path)
            if (path ~ /^".*"$/)
                path = substr(path, 2, length(path) - 2)
            sub(/^[?]/, "", path)
            if (path ~ /^".*"$/)
                path = substr(path, 2, length(path) - 2)
            if (index(path, "~/") == 1)
                path = home substr(path, 2)
            else if (substr(path, 1, 1) != "/")
                path = dir "/" path
            print path
        }
    ' "$1") || ws_die "cannot read $1"
    while IFS= read -r cld_path; do
        [ "${cld_path##*/}" = "${2##*/}" ] || continue
        cld_at=$(cd -P "${cld_path%/*}" 2>/dev/null && pwd -P) || continue
        [ "$cld_at" != "$cld_want" ] || return 0
    done <<EOF
$cld_paths
EOF
    return 1
}

# cld_ghostty_files writes cld.ghostty beside Ghostty's config, and adds the line that reads it at
# the end of that config, once. Ghostty reads a config-file after the file that names it
# (Config.zig's config-file), so window-save-state there wins over the config's own. Ghostty then
# brings its windows back after every quit. It warns, and leaves the config unchanged, where
# cld.ghostty is not its own, the config can't be read, or the config is a link that lacks the line.
# shellcheck disable=SC2154 # header is setup-ssh.sh's, and ws_tmp, lib.sh's, sets ws_tmpfile
cld_ghostty_files() {
    cld_conf=$(cld_ghostty_conf)
    cld_dir=${cld_conf%/*}
    cld_file=$cld_dir/cld.ghostty
    if [ -e "$cld_file" ] || [ -L "$cld_file" ]; then
        case $(head -n 1 "$cld_file" 2>/dev/null) in
        "$header"*) ;;
        *)
            ws_say "warning: $cld_file is not mac/setup.sh's: move it away, as the script" \
                "writes its own"
            return 0
            ;;
        esac
    fi
    if [ -e "$cld_conf" ] && [ ! -r "$cld_conf" ]; then
        ws_say "warning: cannot read $cld_conf, Ghostty's config, so cld.ghostty stays out"
        return 0
    fi
    ws_tmp
    cld_new=$ws_tmpfile
    cat >"$cld_new" <<EOF || ws_die "cannot write $cld_new"
$header: its next run replaces changes made here.
# Ghostty brings back its windows after every quit, which cld restore tabs builds on.
window-save-state = always
EOF
    cld_has=0
    ! cld_includes "$cld_conf" "$cld_file" || cld_has=1
    # A linked directory, as GNU stow's folding makes, gets the warning ssh_files gives for
    # ~/.ssh/config.d.
    if [ -L "$cld_dir" ] && { [ "$cld_has" = 0 ] || ! cmp -s "$cld_file" "$cld_new"; }; then
        ws_say "warning: $cld_dir is a link to $(readlink "$cld_dir"): cld.ghostty and the" \
            "line that reads it go there"
    fi
    [ -d "$cld_dir" ] || ws_dir "$cld_dir" 755
    ws_put "$cld_file" 644 <"$cld_new"
    rm -f "$cld_new"
    [ "$cld_has" = 0 ] || return 0
    if [ -L "$cld_conf" ]; then
        ws_say "warning: $cld_conf is a link to $(readlink "$cld_conf"): add" \
            "'config-file = ?cld.ghostty' at the end of that file"
        return 0
    fi
    cld_mode=644
    [ ! -f "$cld_conf" ] || cld_mode=$(ws_mode "$cld_conf")
    ws_tmp
    cld_new=$ws_tmpfile
    if [ -s "$cld_conf" ]; then
        cat "$cld_conf" >"$cld_new" || ws_die "cannot read $cld_conf"
        [ -z "$(tail -c 1 "$cld_conf")" ] || echo >>"$cld_new"
    fi
    echo 'config-file = ?cld.ghostty' >>"$cld_new" || ws_die "cannot write $cld_new"
    ws_put "$cld_conf" "$cld_mode" <"$cld_new"
    rm -f "$cld_new"
}

# cld_mac sets up cld where Ghostty's app is there: cld itself, its update and Ghostty's settings.
# Then it sets cld_tabs again, which main compares, to write config.d/ws again where it changed.
cld_mac() {
    cld_ghostty || return 0
    if [ -x "$cld_bin" ]; then
        cld_update
    else
        cld_install
    fi
    cld_ghostty_files
    cld_tabs_check
}
