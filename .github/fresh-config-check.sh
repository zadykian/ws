#!/bin/bash
# The playbook job's check of the config Fresh loaded: fresh-config-check.sh CONFIG.JSON SHOW.JSON
# Fresh reads a malformed file with its defaults and exits 0, so a green run alone proves nothing.
set -e
written=$(sed '/^[[:space:]]*\/\//d' "$1" | jq '.keybinding_maps.ws.bindings | length')
jq -e --argjson written "$written" '
    .theme == "JetBrains Dark" and .check_for_updates == false and .self_update == false and
    .auto_attach == false and .active_keybinding_map == "ws" and .editor.rulers == [120] and
    $written > 0 and (.keybinding_maps.ws.bindings | length) == $written' "$2"
