# shellcheck shell=sh
# The function of mac/setup.sh's git step that has git sign with the YubiKey's key, through ws's
# agent. setup-extras.sh sources it after setup-git.sh, whose git_files calls it. It calls the
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
