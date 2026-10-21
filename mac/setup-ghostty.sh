# shellcheck shell=sh
# The part of mac/setup.sh that sets up Ghostty, the Mac's terminal: Ghostty and the font its
# settings name, from Homebrew's casks where missing, and ws.ghostty, those settings, which
# Ghostty's config reads. setup.sh sources it after setup-ssh.sh, whose header it uses, and calls
# ghostty_mac before cld_mac. ghostty_mac warns where Ghostty can't be installed or its files
# written, so that the run goes on to the tunnel. Ghostty then updates itself, as ws.ghostty says.

# 1 where Ghostty's app is there, or where --check would install it, as ghostty_mac sets it.
ghostty_ok=0
# The cask of the font that ws.ghostty names, and the file of its regular style.
ghostty_font=font-jetbrains-mono-nerd-font
ghostty_font_file=JetBrainsMonoNerdFontMono-Regular.ttf

# ghostty_there succeeds where Ghostty's app is there: Ghostty.app in /Applications, where its cask
# puts it, or in ~/Applications.
ghostty_there() {
    [ -d /Applications/Ghostty.app ] || [ -d "$HOME/Applications/Ghostty.app" ]
}

# ghostty_font_there succeeds where the font's regular style is in ~/Library/Fonts, where its cask
# puts it, or in /Library/Fonts.
ghostty_font_there() {
    [ -f "$HOME/Library/Fonts/$ghostty_font_file" ] || [ -f "/Library/Fonts/$ghostty_font_file" ]
}

# ghostty_cask CASK installs Homebrew's CASK. A failed install is a warning, as where brew finds
# files of the cask's that a copy by hand put there. Without Homebrew it stops, as tools does.
ghostty_cask() {
    if ws_checking; then
        ws_say "would run brew install --cask $1"
        return 0
    fi
    ghostty_brew=$(command -v brew) || ws_die "install Homebrew first: https://brew.sh"
    ws_say "installing $1"
    "$ghostty_brew" install --cask "$1" >&2 || ws_say "warning: brew install --cask $1 failed"
}

# ghostty_conf prints the config file that Ghostty's menu opens (src/config/edit.zig, 1.3.1 and
# main). It tries config.ghostty and config in Application Support's com.mitchellh.ghostty, then
# in ~/.config/ghostty. The first that holds anything wins, else the first that exists, else the
# first. Ghostty reads all four. Its app, which launchd starts, sees no XDG_CONFIG_HOME of a
# shell's.
ghostty_conf() {
    ghostty_support="$HOME/Library/Application Support/com.mitchellh.ghostty"
    ghostty_found=
    for ghostty_each in "$ghostty_support/config.ghostty" "$ghostty_support/config" \
        "$HOME/.config/ghostty/config.ghostty" "$HOME/.config/ghostty/config"; do
        if [ -f "$ghostty_each" ] && [ -s "$ghostty_each" ]; then
            echo "$ghostty_each"
            return 0
        fi
        [ -n "$ghostty_found" ] || [ ! -f "$ghostty_each" ] || ghostty_found=$ghostty_each
    done
    echo "${ghostty_found:-$ghostty_support/config.ghostty}"
}

# ghostty_includes CONF FILE succeeds where Ghostty's config CONF has a config-file line that names
# FILE. The line counts with or without the ? and the quotes Ghostty takes, and its path may be
# relative to CONF's directory, under ~/ or full (src/cli/args.zig's LineIterator). A path counts
# by its directory with links resolved, as Ghostty resolves a relative one (src/config/path.zig).
ghostty_includes() {
    [ -f "$1" ] || return 1
    ghostty_want=$(cd -P "${2%/*}" 2>/dev/null && pwd -P) || return 1
    ghostty_paths=$(awk -v dir="${1%/*}" -v home="$HOME" '
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
    while IFS= read -r ghostty_path; do
        [ "${ghostty_path##*/}" = "${2##*/}" ] || continue
        ghostty_at=$(cd -P "${ghostty_path%/*}" 2>/dev/null && pwd -P) || continue
        [ "$ghostty_at" != "$ghostty_want" ] || return 0
    done <<EOF
$ghostty_paths
EOF
    return 1
}

# ghostty_files writes ws.ghostty beside Ghostty's config, mac/ws.ghostty under the header, and
# adds the line that reads it at the end of that config, once. Ghostty reads a config-file after
# the file that names it (Config.zig's config-file), so ws.ghostty's settings win over the
# config's own. It warns, and leaves the config unchanged, where ws.ghostty is not its own, the
# config can't be read, or the config is a link that lacks the line.
# shellcheck disable=SC2154 # header is setup-ssh.sh's, dir setup.sh's, ws_tmpfile lib.sh's
ghostty_files() {
    ghostty_config=$(ghostty_conf)
    ghostty_dir=${ghostty_config%/*}
    ghostty_file=$ghostty_dir/ws.ghostty
    if [ -e "$ghostty_file" ] || [ -L "$ghostty_file" ]; then
        case $(head -n 1 "$ghostty_file" 2>/dev/null) in
        "$header"*) ;;
        *)
            ws_say "warning: $ghostty_file is not mac/setup.sh's: move it away, as the script" \
                "writes its own"
            return 0
            ;;
        esac
    fi
    if [ -e "$ghostty_config" ] && [ ! -r "$ghostty_config" ]; then
        ws_say "warning: cannot read $ghostty_config, Ghostty's config, so ws.ghostty stays out"
        return 0
    fi
    ws_tmp
    ghostty_new=$ws_tmpfile
    cat - "$dir/ws.ghostty" >"$ghostty_new" <<EOF || ws_die "cannot write $ghostty_new"
$header: its next run replaces changes made here.
EOF
    ghostty_has=0
    ! ghostty_includes "$ghostty_config" "$ghostty_file" || ghostty_has=1
    # A linked directory, as GNU stow's folding makes, gets the warning ssh_files gives for
    # ~/.ssh/config.d.
    if [ -L "$ghostty_dir" ] &&
        { [ "$ghostty_has" = 0 ] || ! cmp -s "$ghostty_file" "$ghostty_new"; }; then
        ws_say "warning: $ghostty_dir is a link to $(readlink "$ghostty_dir"): ws.ghostty and" \
            "the line that reads it go there"
    fi
    [ -d "$ghostty_dir" ] || ws_dir "$ghostty_dir" 755
    ws_put "$ghostty_file" 644 <"$ghostty_new"
    rm -f "$ghostty_new"
    [ "$ghostty_has" = 0 ] || return 0
    if [ -L "$ghostty_config" ]; then
        ws_say "warning: $ghostty_config is a link to $(readlink "$ghostty_config"): add" \
            "'config-file = ?ws.ghostty' at the end of that file"
        return 0
    fi
    ghostty_mode=644
    [ ! -f "$ghostty_config" ] || ghostty_mode=$(ws_mode "$ghostty_config")
    ws_tmp
    ghostty_new=$ws_tmpfile
    if [ -s "$ghostty_config" ]; then
        cat "$ghostty_config" >"$ghostty_new" || ws_die "cannot read $ghostty_config"
        [ -z "$(tail -c 1 "$ghostty_config")" ] || echo >>"$ghostty_new"
    fi
    echo 'config-file = ?ws.ghostty' >>"$ghostty_new" || ws_die "cannot write $ghostty_new"
    ws_put "$ghostty_config" "$ghostty_mode" <"$ghostty_new"
    rm -f "$ghostty_new"
}

# ghostty_mac installs Ghostty where its app is missing, then its font where missing, and writes
# ws.ghostty. It sets ghostty_ok, which cld_mac reads, to 1 where the app is there, or where
# --check would install it.
# shellcheck disable=SC2034 # setup-cld.sh's cld_mac reads ghostty_ok
ghostty_mac() {
    ghostty_ok=0
    ghostty_there || ghostty_cask ghostty
    ghostty_there || ws_checking || return 0
    ghostty_ok=1
    ghostty_font_there || ghostty_cask "$ghostty_font"
    ghostty_files
}
