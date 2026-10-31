#!/bin/sh
# The stand-ins of the mac and mac-wizard jobs (.github/workflows/mac.yml, mac-wizard/standin.sh),
# made before their first run of mac/setup.sh. They spare each run the downloads, which mac-cld.yml
# makes for real, so that the jobs' many runs that must print nothing touch no network:
#
#   sh .github/mac-standins.sh
#
# - JetBrains Gateway.app and Ghostty.app, empty directories in ~/Applications, and the regular
#   style of Ghostty's font, an empty file in ~/Library/Fonts, which spare their casks. setup.sh
#   writes ws.ghostty, as on a Mac with Ghostty;
# - ~/.local/bin/cld, whose version names no release, so that setup.sh neither installs nor updates
#   it, as a cld built from a clone. It lacks restore tabs, so ssh's hosts get no LocalCommand.

set -eu

if [ -e ~/.local/bin/cld ]; then
    echo '::error::the runner has ~/.local/bin/cld already'
    exit 1
fi
mkdir -p ~/Applications/"JetBrains Gateway.app" ~/Applications/Ghostty.app ~/Library/Fonts \
    ~/.local/bin
: >~/Library/Fonts/JetBrainsMonoNerdFontMono-Regular.ttf
cat >~/.local/bin/cld <<'EOS'
#!/bin/sh
# A stand-in cld: it answers --version with no release's, and fails at anything else.
[ "$*" = --version ] && echo 'cld stand-in'
EOS
chmod 755 ~/.local/bin/cld
