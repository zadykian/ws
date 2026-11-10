#!/bin/bash
# ws mac apply writes ~/.config/ws/zshrc, mac/zshrc whole, and ends ~/.zshrc with the line that
# sources it. An interactive zsh prints the resets before each prompt, and the alternate screen's
# reset after a 255 from ssh, or from cld with --host, alone. Functions stand in for both.

set -eo pipefail

fail=0
cmp -s ~/.config/ws/zshrc mac/zshrc ||
    {
        echo '::error::~/.config/ws/zshrc is not mac/zshrc'
        fail=1
    }
[ "$(tail -n 1 ~/.zshrc)" = 'source ~/.config/ws/zshrc' ] ||
    {
        echo '::error::~/.zshrc does not end with the line that sources the hook'
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
