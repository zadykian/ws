-- The plugin calls `ws unit` once per hook, with every unit the hook is given, and the write hook
-- has ws verify mise's plan first. ws finds its side's specs itself, and holds every check: this
-- file passes names and states alone.
local cmd = require("cmd")

local M = {}

local function quote(s)
    return "'" .. s:gsub("'", "'\\''") .. "'"
end

-- binary is the pinned ws, where mise installed it, or nil before mise's tools step installs it.
-- It asks mise alone: a /usr/local/bin/ws of an older pin would win on PATH.
function M.binary()
    local out = cmd.exec("mise which ws 2>/dev/null || true")
    return out:match("(%S+)")
end

local function command(bin, verb, pkgs, flags)
    local line = { quote(bin), "unit", verb, flags or "" }
    for _, pkg in ipairs(pkgs) do
        table.insert(line, quote(pkg.name))
    end
    return table.concat(line, " ")
end

-- read returns each unit's state by name, from a line per unit: NAME, a tab, the state.
function M.read(bin, pkgs)
    local states = {}
    for name, state in cmd.exec(command(bin, "read", pkgs)):gmatch("([^\t\n]+)\t([^\n]*)") do
        states[name] = state
    end
    return states
end

-- run runs ws's command line, streaming its lines, and fails where ws fails.
local function run(line, verb)
    local code = os.execute(line)
    if code ~= 0 and code ~= true then
        error("ws unit " .. verb .. " exited " .. tostring(code))
    end
end

-- write streams ws's lines, the diffs of a dry run among them, and fails where ws fails. ws unit
-- verify runs first. On a first run, and on the run that brings a new pin, the files' check runs
-- before mise installs that ws, and passes unchecked. Each unit reads as missing then, so mise
-- runs this hook with them, and the check runs here.
function M.write(pkgs, dry_run)
    local bin = M.binary()
    if not bin then
        error("ws is not installed: mise's tools step installs it")
    end
    run(command(bin, "verify", {}), "verify")
    run(command(bin, "write", pkgs, dry_run and "--dry-run" or ""), "write")
end

return M
