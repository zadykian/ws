# shellcheck shell=sh
# The functions of mac/setup.sh's git step that have git sign with the YubiKey's key, through ws's
# agent. setup-extras.sh sources it after setup-git.sh, whose git_files calls them. They call the
# functions of lib.sh and setup-lib.sh.

# git_agent writes ~/.config/ws/ws-ssh-keygen, git's gpg.ssh.program for the YubiKey's key, and
# sets git_program to it. The wrapper gives ssh-keygen ws's agent, and the agent's keys alone (-U):
# with the YubiKey out, git signs nothing, though a key in a file sits beside the .pub.
# shellcheck disable=SC2154 # git_dir is setup-git.sh's, and ws_agent_socket setup-conf.sh's
git_agent() {
    git_program=$git_dir/ws-ssh-keygen
    ws_put "$git_program" 755 <<EOF
#!/bin/sh
# Written by ws's mac/setup.sh: git's gpg.ssh.program, ssh-keygen with ws's agent, which holds the
# YubiKey's keys, and with no key of a file. Its next run replaces changes made here.
SSH_AUTH_SOCK='$ws_agent_socket'
export SSH_AUTH_SOCK
exec ssh-keygen -U "\$@"
EOF
}

# git_secretive removes ~/.config/ws/secretive-ssh-keygen, git's gpg.ssh.program for a key of
# Secretive's, which older runs wrote and gitconfig names no more. A file there whose second line
# is not setup.sh's stays. Under --check, it shows the removal as a diff.
git_secretive() {
    git_old=$git_dir/secretive-ssh-keygen
    [ -f "$git_old" ] || return 0
    case $(sed -n 2p "$git_old") in
    "# Written by ws's mac/setup.sh"*) ;;
    *) return 0 ;;
    esac
    if ws_checking; then
        ws_diff "$git_old" /dev/null "$git_old"
    else
        rm -f "$git_old" || ws_die "cannot remove $git_old"
        ws_say "removed $git_old"
    fi
}
