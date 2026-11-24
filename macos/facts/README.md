# The Mac's facts

The Mac's specs, each role's `units.toml.tmpl` under `macos/roles`, are filled with these facts.
`ws unit` on the Mac reads them before it loads a unit:

| Fact | Its meaning |
| --- | --- |
| `.User`, `.Home` | The account that ws runs as and its home, from `USER` and `HOME`. |
| `.Servers` | Each server of `servers/` beside `values.yml`, by name: its `.Name` and its values. |
| `.Secrets.FnoxFrom`, `.Secrets.PluginFrom` | fnox and age-plugin-yubikey as mise installed them for the tree, by `mise -C /opt/ws/macos which`, or empty where mise has none. |
| `.Ghostty.Config` | The config that Ghostty's menu opens: the first of its four that holds anything, else the first there, else its default. |

Each server's values are `values.Conf`'s fields, the Mac's own keys joined in, as
[the values](../values/README.md) say. The read runs the whole check of the values, so that a bad
value, a bad name or a clash between two servers fails every unit's read. mise's plan then stops
before a change. A Mac with no server yet has no `.Servers`.
