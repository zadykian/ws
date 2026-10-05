# ws

Sets up my development server from code. Ansible runs on the machine itself and installs packages,
Docker, the shell, tmux, the helix editor, claude and [cld](https://github.com/zadykian/cld).
Running it again is safe, and `--check --diff` shows how a machine differs from the repository.

Nothing is here yet: the roles land one pull request at a time.

## By hand

Secrets are never in this repository. On a new machine, these stay manual:

- `gh auth login` and `gh auth setup-git`;
- the claude login.

## Pull requests

Pull requests land on `main` by fast-forward only, as the commits CI checked. GitHub's merge
methods are all refused. A comment `/fast-forward` on a pull request pushes its head to `main`
(`.github/workflows/fast-forward.yml`). A pull request that changes `.github/workflows` is pushed
by hand, `git push origin SHA:main`.

## License

[MIT](LICENSE)
