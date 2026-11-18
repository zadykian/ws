#!/bin/sh
# The stand-ins of the mac job (.github/workflows/mac.yml),
# made before their first run of ws client macos apply. They spare each run the downloads, which
# mac-cld.yml makes for real, so that the jobs' many runs that must print nothing touch no network:
#
#   sh .github/mac-standins.sh
#
# - Ghostty.app and Zed.app, empty directories in ~/Applications, and the regular style of their
#   font, an empty file in ~/Library/Fonts. mise bootstrap's casks (28-bootstrap.sh) install the
#   apps; the stand-in of Ghostty's app keeps cld's step, which looks for it, off brew;
# - wireguard-tools from Homebrew where missing, whose wg and wireguard-go make the stand-in ws.
#   The daemon needs neither, so this is the one download, a small formula, that the jobs make;
# - ~/.local/bin/cld, whose version names no release, so that ws client macos apply neither installs
#   nor updates it, as a cld built from a clone. It lacks restore tabs, so ssh's hosts get no
#   LocalCommand.

set -eu

if [ -e ~/.local/bin/cld ]; then
    echo '::error::the runner has ~/.local/bin/cld already'
    exit 1
fi
brew list --formula wireguard-tools >/dev/null 2>&1 || brew install wireguard-tools
mkdir -p ~/Applications/Ghostty.app ~/Applications/Zed.app ~/Library/Fonts ~/.local/bin
: >~/Library/Fonts/JetBrainsMonoNerdFontMono-Regular.ttf
cat >~/.local/bin/cld <<'EOS'
#!/bin/sh
# A stand-in cld: it answers --version with no release's, and fails at anything else.
[ "$*" = --version ] && echo 'cld stand-in'
EOS
chmod 755 ~/.local/bin/cld
