# ws

Sets up my development server from code. Ansible runs on the machine itself and installs packages,
Docker, the shell, tmux, the helix editor, claude and [cld](https://github.com/zadykian/cld).
Running it again is safe, and `--check --diff` shows how a machine differs from the repository.

The roles land one pull request at a time. Each is a tag of `site.yml`:

- `base`: the packages every machine gets, the locale `en_US.UTF-8`, the timezone UTC,
  unattended-upgrades as Ubuntu ships it, which installs Ubuntu's security updates daily, and a
  swap file, `/swapfile`, the size of the RAM rounded up to a whole GiB. `-e base_swap_size_mb=N`
  sets another size in MiB, and 0 none. The file replaces the installer's `/swap.img`. A run
  stops before it changes the swap where the disk lacks room for the file and 2 GiB more, or
  where swapping an area off would bring back more than half the memory available.

## A new machine

Install Ubuntu Server 26.04 by hand from its installer, with an SSH key to log in with. Then run
`bootstrap.sh` as root, after `sudo -i` if you log in as the installer's user:

```sh
curl -fsSL https://raw.githubusercontent.com/zadykian/ws/main/bootstrap.sh | sh
```

It installs ansible-core and git with apt, clones this repository into `/root/repository/ws`,
installs the Ansible collections of `requirements.yml`, and runs `site.yml`. Then do what stays
by hand, below.

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
