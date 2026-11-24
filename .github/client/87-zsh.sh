#!/bin/bash
# mise bootstrap (28-bootstrap.sh) wrote /etc/ws/zshrc, as root, macos/roles/zsh/files/zshrc
# whole, and ended ~/.zshrc with the line that sources it. It cloned oh-my-zsh and its plugins at
# the commits of macos/roles/zsh/mise.macos.toml, which an interactive zsh loads. That zsh prints
# the resets before each prompt, and the alternate screen's reset after a 255 from ssh, or from
# cld with --host, alone. Functions stand in for both.

set -eo pipefail

fail=0
cmp -s /etc/ws/zshrc macos/roles/zsh/files/zshrc ||
    {
        echo '::error::/etc/ws/zshrc is not macos/roles/zsh/files/zshrc'
        fail=1
    }
[ "$(tail -n 1 ~/.zshrc)" = 'source /etc/ws/zshrc' ] ||
    {
        echo '::error::~/.zshrc does not end with the line that sources the hook'
        fail=1
    }

# pin PATH is the commit that mise.macos.toml names for ~/PATH, and the clone's head.
pin() {
    want=$(grep -F "\"~/$1\"" macos/roles/zsh/mise.macos.toml | sed -E 's/.*ref = "([0-9a-f]{40})".*/\1/')
    [ -n "$want" ] && [ "$(git -C ~/"$1" rev-parse HEAD)" = "$want" ] ||
        {
            echo "::error::~/$1 is not at the commit of macos/roles/zsh/mise.macos.toml"
            fail=1
        }
}
pin .oh-my-zsh
pin .oh-my-zsh/custom/plugins/zsh-autosuggestions
pin .oh-my-zsh/custom/plugins/zsh-syntax-highlighting

# An interactive zsh with only a home loads the git plugin and the two others. It keeps oh-my-zsh's
# update off, and has ~/.local/bin first on PATH.
# shellcheck disable=SC2016 # zsh expands the script's variables, not bash
env -i HOME="$HOME" zsh -ic '
    alias gst >/dev/null && alias genpass >/dev/null &&
        whence -w _zsh_autosuggest_start >/dev/null && whence -w _zsh_highlight >/dev/null &&
        zstyle -s ":omz:update" mode mode && [[ $mode == disabled ]] &&
        [[ ${path[1]} == $HOME/.local/bin ]]' </dev/null ||
    {
        echo '::error::zsh -i lacks a plugin, an alias, the update setting or ~/.local/bin first on PATH'
        fail=1
    }

# prompt COMMAND SCREENS runs COMMAND in zsh -i, and checks that the resets come before each of its
# three prompts, the stand-ins' line, COMMAND and exit. The alternate screen's reset, after ESC 7
# and the resets, comes SCREENS times, and the resets once more after each.
prompt() {
    out=$(printf 'ssh() { return 255; }; cld() { return 255; }\n%s\nexit 0\n' "$1" | zsh -i 2>/dev/null)
    esc=$(printf '\033')
    modes=$(grep -o "$esc\\[?2031l" <<<"$out" | wc -l || :)
    screens=$(grep -o "$esc\\[?25h${esc}7$esc\\[?1049l$esc" <<<"$out" | wc -l || :)
    [ "$((modes))" = $((3 + $2)) ] && [ "$((screens))" = "$2" ] ||
        {
            echo "::error::zsh -i on '$1': $((modes)) resets and $((screens)) of the alternate" \
                "screen; want $((3 + $2)) and $2"
            fail=1
        }
}
prompt 'ssh ws' 1
prompt 'cld join --host ws -n s1' 1
prompt 'cld list' 0
prompt '(exit 255)' 0
prompt true 0
exit "$fail"
