#!/bin/sh
# The wizard's ssh to ws runs no LocalCommand, on the Mac that 31-host.sh leaves. A Host * block in
# ~/.ssh/config permits one and has it touch a marker. A plain ssh to ws must then touch it, and a
# run of the wizard, whose calls to ws pass PermitLocalCommand=no, must not.

set -u

wsbin=$HOME/.local/bin/ws
marker=$RUNNER_TEMP/localcommand
fail=0

cp ~/.ssh/config "$RUNNER_TEMP/ssh.config"
{
    printf '%s\n' 'Host *' '    PermitLocalCommand yes'
    printf '    LocalCommand touch %s\n' "$marker"
} >>~/.ssh/config
rm -f "$marker"
ssh -o BatchMode=yes -o HostName=localhost -o Port=22022 ws true </dev/null || echo "::warning::ssh to ws true exited $?"
if [ ! -e "$marker" ]; then
    echo '::error::a plain ssh to ws ran no LocalCommand, so this case proves nothing'
    fail=1
fi
rm -f "$marker"
out=$("$wsbin" client macos wizard </dev/null 2>&1)
status=$?
printf '%s\n' "$out"
if [ "$status" != 0 ]; then
    echo "::error::ws client macos wizard exited $status"
    fail=1
fi
if [ -e "$marker" ]; then
    echo '::error::the wizard'"'"'s ssh to ws ran a LocalCommand'
    fail=1
fi
cp "$RUNNER_TEMP/ssh.config" ~/.ssh/config
rm -f "$marker" "$RUNNER_TEMP/ssh.config"
exit "$fail"
