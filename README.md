# ws

Sets up my development server from code. Ansible runs on the machine itself and sets up packages,
SSH by key and a firewall, Docker, the shell, tmux, the helix editor, claude and
[cld](https://github.com/zadykian/cld).
Running it again is safe, and `--check --diff` shows how a machine differs from the repository.

The roles land one pull request at a time. Each is a tag of `site.yml`:

- `base`: the packages every machine gets, the locale `en_US.UTF-8`, the timezone UTC,
  unattended-upgrades as Ubuntu ships it, which installs Ubuntu's security updates daily, and a
  swap file, `/swapfile`, the size of the RAM rounded up to a whole GiB. `-e base_swap_size_mb=N`
  sets another size in MiB, and 0 none. The file replaces the installer's `/swap.img`. A run
  stops before it changes the swap where the disk lacks room for the file and 2 GiB more, or
  where swapping an area off would bring back more than half the memory available.
- `security`: an sshd drop-in that lets everyone log in by key only, ufw, and fail2ban for SSH:
  [SSH and the firewall](#ssh-and-the-firewall).
- `shell`: `~/.bashrc` with ble.sh and bash-completion, `PATH` in `~/.profile`, and
  `~/.gitconfig`; see [The shell](#the-shell).
- `tmux`: tmux from Ubuntu's archive, with no config, as cld runs tmux with `-f /dev/null`. Where
  tmux's snap is installed, the role removes it. Where cld's sessions run the snap's tmux, end them
  before a real run: cld then finds `/usr/bin/tmux` first on the `PATH`, and their key bindings
  call `/snap/bin/tmux`, which the removal takes away.

## A new machine

Install Ubuntu Server 26.04 by hand from its installer, with an SSH key to log in with. The
installer gives the key to its own user, and the playbook lets root log in by key only, so copy
the key to root first, as the installer's user:

```sh
sudo install -d -m 700 /root/.ssh
sudo install -m 600 ~/.ssh/authorized_keys /root/.ssh/authorized_keys
```

Then run `bootstrap.sh` as root, after `sudo -i` if you log in as the installer's user:

```sh
curl -fsSL https://raw.githubusercontent.com/zadykian/ws/main/bootstrap.sh | sh
```

It installs ansible-core and git with apt, clones this repository into `/root/repository/ws`,
installs the Ansible collections of `requirements.yml`, and runs `site.yml`. Then do what stays
by hand, below.

## SSH and the firewall

The playbook turns SSH's passwords off: everyone logs in by key, root included. ufw then lets in
SSH on port 22 and nothing else, and fail2ban bans an address for 10 minutes after 5 failed logins
in 10 minutes. Over SSH, keep the session that ran the playbook open until a new one logs in.
The playbook stops before it turns passwords off if root has no key to log in with.

Docker's published ports get past ufw: publish a container's port on `127.0.0.1` and reach it
through an SSH tunnel.

## Running it again

`bootstrap.sh` leaves the clone in `/root/repository/ws` as it is, so pull first:

```sh
git -C /root/repository/ws pull --ff-only
/root/repository/ws/bootstrap.sh
```

Its arguments go to `ansible-playbook`: `--tags NAME` runs the role NAME alone.

## Dry runs

With `--check --diff`, the playbook shows what it would change and changes nothing:

```sh
/root/repository/ws/bootstrap.sh --check --diff
```

On a new machine, that is:

```sh
curl -fsSL https://raw.githubusercontent.com/zadykian/ws/main/bootstrap.sh | sh -s -- --check --diff
```

`bootstrap.sh` still installs ansible-core and git, clones the repository and installs the
collections, as the playbook cannot run without them.

## By hand

Secrets are never in this repository. On a new machine, these stay manual:

- `gh auth login` and `gh auth setup-git`;
- the claude login.

## The shell

The `shell` role writes `~/.bashrc`, `~/.profile` and `~/.gitconfig`, and each run replaces any
change made to them by hand. A machine's own settings go in files they read where they exist:

- `~/.secrets/env`, which `~/.bashrc` sources first, above its interactive guard, so that
  non-interactive shells get it too;
- `~/.bash_aliases`, which `~/.bashrc` sources too;
- `~/.config/git/local`, which `~/.gitconfig` includes last: a signing key, say.

An installer that appends to `~/.bashrc` or `~/.profile`, a line that adds its directory to
`PATH`, say, loses that line on the next run: move it to `~/.bash_aliases`.

`~/.gitconfig` has gh's credential helper already, as `gh auth setup-git` writes it, so running
that changes nothing. A dry run shows no diff of `~/.gitconfig`, as one set up by hand may hold a
credential. It names instead the keys a real run would drop, without their values: move them to
`~/.config/git/local` first. A subsection shows as `*`, as in `http.*.extraheader`, since one may
be a URL, which names a host. A real run keeps the old file beside the new one, as
`~/.gitconfig.*~`.

## Checks

CI runs yamllint, ansible-lint with its production profile, ShellCheck and shfmt on the shell
scripts, and gitleaks over the whole history. Any finding fails it, warnings included.

CI also runs `bootstrap.sh` in an Ubuntu 26.04 container, without the tasks tagged `systemd`,
which need a booted machine. It runs it twice, and the second run must change nothing. Its swap
file is 64 MiB, as the runner's disk has no room for one the size of its RAM.

The hooks in `.githooks` check each commit before it is made. Enable them in a clone, and so in
its worktrees, with `git config core.hooksPath .githooks`, as `bootstrap.sh` does in its clone.
Where `~/.config/ws/forbidden-words` exists, a word or phrase a line, they refuse a commit whose
added lines, file names or message hold one. Where gitleaks is installed, they refuse a staged
secret too.

## Pull requests

Pull requests land on `main` by fast-forward only, as the commits CI checked. GitHub's merge
methods are all refused. A comment `/ff` on a pull request pushes its head to `main`
(`.github/workflows/fast-forward.yml`). A pull request that changes `.github/workflows` is pushed
by hand, `git push origin SHA:main`.

## License

[MIT](LICENSE)
