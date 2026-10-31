# shellcheck shell=sh
# The part of mac/setup.sh that sets up cld, for cld restore tabs, which reopens Ghostty's tabs of
# cld's sessions. setup.sh sources it after setup-ghostty.sh, whose functions it uses, and calls
# cld_mac after ghostty_mac. cld_mac installs and updates cld
# wherever setup.sh runs. The lines of ssh's hosts need Ghostty's app, or under --check its install.
# cld_mac warns where cld can't be set up, so that the run goes on to the tunnel.

# cld where its install.sh puts it, and the releases whose latest has that script.
cld_bin=$HOME/.local/bin/cld
cld_releases=https://github.com/zadykian/cld/releases
# 1 where ssh's hosts start cld's watcher, as cld_tabs_check sets it, and 1 once --check noted
# that they may come.
cld_tabs=0
cld_noted=0

# cld_tabs_check sets cld_tabs to 1 where Ghostty's app is there and the cld in ~/.local/bin has
# restore tabs, else 0. cld help restore tabs fails in a cld without it, so that ssh never runs a
# cld that would refuse restore tabs --watch at each connection.
cld_tabs_check() {
    cld_tabs=0
    if ghostty_there && [ -x "$cld_bin" ] && "$cld_bin" help restore tabs >/dev/null 2>&1; then
        cld_tabs=1
    fi
}

# cld_would_tabs says, under --check, that config.d/ws would gain the lines that start cld's
# watcher where ghostty_mac found Ghostty's app, or would install it, and the cld to come has
# restore tabs. --check can run neither a cld nor a Ghostty that it would install. It says so once.
# shellcheck disable=SC2154 # ghostty_ok is setup-ghostty.sh's
cld_would_tabs() {
    [ "$cld_tabs" = 0 ] && [ "$ghostty_ok" = 1 ] && [ "$cld_noted" = 0 ] || return 0
    cld_noted=1
    ws_say "would add the lines that start cld's watcher to config.d/ws," \
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

# cld_mac sets up cld, itself and its update, wherever setup.sh runs. Then it sets cld_tabs again,
# which main compares, to write config.d/ws again where it changed. Under --check, a cld there may
# have restore tabs already, while ghostty_mac would install Ghostty's app: the lines come then too.
cld_mac() {
    if [ -x "$cld_bin" ]; then
        cld_update
    else
        cld_install
    fi
    cld_tabs_check
    ! ws_checking || ! "$cld_bin" help restore tabs >/dev/null 2>&1 || cld_would_tabs
}
