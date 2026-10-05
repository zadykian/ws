# ws

Sets up my development server from code. Ansible runs on the machine itself and installs packages,
Docker, the shell, tmux, the helix editor, claude and [cld](https://github.com/zadykian/cld).
Running it again is safe, and `--check --diff` shows how a machine differs from the repository.

Nothing is here yet: the roles land one pull request at a time.

## By hand

Secrets are never in this repository. On a new machine, these stay manual:

- `gh auth login` and `gh auth setup-git`;
- the claude login.

## Checks

CI runs yamllint, ansible-lint with its production profile, ShellCheck and shfmt on the shell
scripts, and gitleaks over the whole history. Any finding fails it, warnings included.

The hooks in `.githooks` check each commit before it is made. Enable them in a clone, and so in
its worktrees, with `git config core.hooksPath .githooks`. Where `~/.config/ws/forbidden-words`
exists, a word or phrase a line, they refuse a commit whose added lines, file names or message
hold one. Where gitleaks is installed, they refuse a staged secret too.

## Pull requests

Pull requests land on `main` by fast-forward only, as the commits CI checked. GitHub's merge
methods are all refused. A comment `/ff` on a pull request pushes its head to `main`
(`.github/workflows/fast-forward.yml`). A pull request that changes `.github/workflows` is pushed
by hand, `git push origin SHA:main`.

## License

[MIT](LICENSE)
