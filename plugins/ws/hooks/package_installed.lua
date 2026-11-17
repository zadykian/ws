local ws = require("ws")

-- Each unit reports "in-sync" or its pending change, which mise shows as an update. Without ws each
-- is missing, which mise shows as a create. A unit whose plan fails fails the hook, so that the
-- plan exits 1, not 2.
function PLUGIN:PackageInstalled(ctx)
    local bin = ws.binary()
    local states = bin and ws.read(bin, ctx.packages) or {}
    local out, failed = {}, {}
    for _, pkg in ipairs(ctx.packages) do
        local state = states[pkg.name]
        if state and state:match("^fails: ") then
            table.insert(failed, pkg.name .. " " .. state)
        end
        table.insert(
            out,
            { name = pkg.name, state = bin and "installed" or "missing", version = state }
        )
    end
    if #failed > 0 then
        error(table.concat(failed, "; "))
    end
    return { packages = out }
end
