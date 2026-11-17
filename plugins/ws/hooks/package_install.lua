local ws = require("ws")

function PLUGIN:PackageInstall(ctx)
    ws.write(ctx.packages, ctx.dry_run)
end
