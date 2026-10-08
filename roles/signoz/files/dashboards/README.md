# The dashboards

The role keeps a dashboard in SigNoz for each file here that `signoz_dashboards` names: `host.json`,
shown as Host, and `claude.json`, shown as Claude. It finds each by the fixed `name` in its file.

## Host

`host.json` is SigNoz's Host Metrics dashboard, `hostmetrics/hostmetrics.json` in
[SigNoz/dashboards](https://github.com/SigNoz/dashboards) at commit `1fd19dd`, under the Apache
License 2.0 in [LICENSE](LICENSE), which covers `host.json` alone. This repository changed it:

- Its name is `host`, by which the role finds it, and it shows as Host, with a description of its
  own.
- The File system usage panel averages `system.filesystem.usage`, as the inode panel does. Its
  rate, which the original took, gives no rows: the collector sends the metric as a sum that goes
  up and down.
- A row of panels shows temperatures and fans from node_exporter: `node_hwmon_temp_celsius`,
  `node_hwmon_fan_rpm` and `node_thermal_zone_temp`.

## Claude

`claude.json` was written for this repository, under its MIT license. It reads what claude sends
to the collector, as the `claude` role sets it:

- The metrics `claude_code.cost.usage`, `.token.usage`, `.session.count`, `.active_time.total`,
  `.lines_of_code.count`, `.commit.count`, `.pull_request.count` and `.code_edit_tool.decision`,
  each by the attributes claude gives it, such as `model`, `type`, `query_source`, `start_type`,
  `tool_name`, `decision` and `source`. Sessions leave out `start_type` `agents_view`, the
  `claude agents` view.
- The events `user_prompt`, `tool_result` and `api_error`, from SigNoz's logs whose `service.name`
  is `claude-code`, found by their body, as `claude_code.user_prompt`. Their `event.name` attribute
  won't do: SigNoz's filter takes `event.` for a context, as it does `resource.`, and finds no key
  `name`. `status_code` is a number, and an error with none, such as a refused connection, shows
  without one.

Each metric's panel takes its `increase` over time and sums it across series. claude sends its
counters as deltas, which the `claude` role leaves as they are, and that gives their totals. A
claude set to send them as cumulative sums would show less. SigNoz counts no increase in the first
step of a cumulative series, and each session is a series of its own.

The bars are days in UTC, the server's timezone. Today's bar can lack the last five minutes.
SigNoz's query cache reads them apart from the rest of the day, and keeps the earlier part alone
for a bar that began before them. Cost is claude's estimate at API prices, which under a
subscription is not what was billed. No panel shows a prompt's text, a tool's input or a command.
The dashboard has no variable, as one account reports here: every point carries `user.email` and
the account's IDs, by which one could filter later.

## Refreshing them

Each file is kept as SigNoz's API returns it, with the empty fields SigNoz adds, so that the role
can compare the two. A SigNoz that returns other fields would have the role replace a dashboard on
every run. After a run on the new SigNoz, refresh both files from the dashboards SigNoz holds, in
this directory:

```sh
url=http://127.0.0.1:3301/api/v2/dashboards
for name in host claude; do
    id=$(curl -s "$url?sort=created_at&order=asc&limit=200" |
        jq -r --arg name "$name" '[.data.dashboards[] | select(.name == $name)][0].id')
    curl -s "$url/$id" | jq '.data | {schemaVersion, image, name, tags, spec}' >"$name.json"
done
```

The list gives 200 dashboards at most: where there are more, add `&offset=200` and so on.
