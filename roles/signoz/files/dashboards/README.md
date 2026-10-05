# The Host dashboard

`host.json` is SigNoz's Host Metrics dashboard, `hostmetrics/hostmetrics.json` in
[SigNoz/dashboards](https://github.com/SigNoz/dashboards) at commit `1fd19dd`, under the Apache
License 2.0 in [LICENSE](LICENSE). It was changed for this repository:

- It is named `host`, by which the role finds it, and shows as Host, with a description of its own.
- The File system usage panel averages `system.filesystem.usage`, as the inode panel does. Its
  rate, which the original took, gives no rows: the collector sends the metric as a sum that goes
  up and down.
- A row of panels shows temperatures and fans from node_exporter: `node_hwmon_temp_celsius`,
  `node_hwmon_fan_rpm` and `node_thermal_zone_temp`.
- It is kept as SigNoz's API returns it, with the empty fields SigNoz adds, so that the role can
  compare the two.

A SigNoz that returns other fields would have the role replace the dashboard on every run. After a
run on the new SigNoz, refresh the file from the dashboard SigNoz holds, in this directory:

```sh
url=http://127.0.0.1:3301/api/v2/dashboards
id=$(curl -s "$url?sort=created_at&order=asc&limit=200" |
    jq -r '[.data.dashboards[] | select(.name == "host")][0].id')
curl -s "$url/$id" | jq '.data | {schemaVersion, image, name, tags, spec}' >host.json
```

The list gives 200 dashboards at most: where there are more, add `&offset=200` and so on.
