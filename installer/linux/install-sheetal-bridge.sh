#!/usr/bin/env bash
set -euo pipefail

SERVER="${SHEETAL_SERVER:-https://sd.vishalojha.me}"
BASE="${XDG_CONFIG_HOME:-$HOME/.config}/sheetal-bridge"
BIN="$HOME/.local/bin/sheetal-bridge"
mkdir -p "$BASE" "$HOME/.local/bin" "$HOME/.config/systemd/user"
read -r -p "Paste your Sheetal bridge token: " TOKEN
if [[ -z "$TOKEN" ]]; then echo "A bridge token is required." >&2; exit 2; fi
curl -fL "$SERVER/downloads/sheetal-bridge-linux-amd64" -o "$BIN"
chmod 700 "$BIN"
printf '{"server":"%s","token":"%s","device":"linux-laptop"}\n' "$SERVER" "$TOKEN" > "$BASE/config.json"
chmod 600 "$BASE/config.json"
cat > "$HOME/.config/systemd/user/sheetal-bridge.service" <<EOF
[Unit]
Description=Sheetal local browser bridge
After=network-online.target

[Service]
ExecStart=$BIN
Restart=always
RestartSec=3

[Install]
WantedBy=default.target
EOF
systemctl --user daemon-reload
systemctl --user enable --now sheetal-bridge.service
echo "Sheetal Bridge is installed and running."
