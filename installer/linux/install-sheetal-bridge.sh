#!/usr/bin/env bash
set -euo pipefail

SERVER="${SHEETAL_SERVER:-https://sd.vishalojha.me}"
BASE="${XDG_CONFIG_HOME:-$HOME/.config}/agent-v"
BIN="$HOME/.local/bin/agent-v"
mkdir -p "$BASE" "$HOME/.local/bin" "$HOME/.config/systemd/user"
systemctl --user disable --now sheetal-bridge.service 2>/dev/null || true
rm -f "$HOME/.config/systemd/user/default.target.wants/sheetal-bridge.service"
read -r -p "Paste your Agent V token (leave blank if not enabled): " TOKEN
curl -fL "$SERVER/downloads/agent-v-linux-amd64" -o "$BIN"
chmod 700 "$BIN"
printf '{"server":"%s","token":"%s","device":"linux-laptop"}\n' "$SERVER" "$TOKEN" > "$BASE/config.json"
chmod 600 "$BASE/config.json"
cat > "$HOME/.config/systemd/user/agent-v.service" <<EOF
[Unit]
Description=Agent V local browser control
After=network-online.target

[Service]
ExecStart=$BIN
Restart=always
RestartSec=3

[Install]
WantedBy=default.target
EOF
systemctl --user daemon-reload
systemctl --user enable --now agent-v.service
echo "Agent V is installed and running."
