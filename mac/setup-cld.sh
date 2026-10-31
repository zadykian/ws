# shellcheck shell=sh
# The part of mac/setup.sh that sets up cld, for cld restore tabs, which reopens Ghostty's tabs of
# cld's sessions. setup.sh sources it after setup-ssh.sh and calls cld_mac after extras. cld_mac
# warns where cld can't be set up, so that the run goes on to the tunnel.

# cld where its install.sh puts it, and the releases whose latest has that script.
cld_bin=$HOME/.local/bin/cld
cld_releases=https://github.com/zadykian/cld/releases

# cld_install installs cld into ~/.local/bin, replacing any cld there, with the install.sh of cld's
# latest release, as the claude role does on ws. install.sh checks the binary against the
# release's cld.sha256 before it moves it there. It edits no shell profile.
# shellcheck disable=SC2154 # ws_tmp, lib.sh's, sets ws_tmpfile
cld_install() {
    if ws_checking; then
        ws_say "would install cld into ~/.local/bin with the install.sh of its latest release"
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

# cld_mac sets up cld: it installs it where it is missing.
cld_mac() {
    [ -x "$cld_bin" ] || cld_install
}
