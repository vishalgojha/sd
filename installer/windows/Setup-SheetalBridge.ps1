param([string]$ExePath = "$PSScriptRoot\SheetalBridge.exe", [string]$Server = "https://sd.vishalojha.me")
$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName Microsoft.VisualBasic
$app = Join-Path $env:APPDATA "AgentV"
New-Item -ItemType Directory -Force -Path $app | Out-Null
Copy-Item $ExePath (Join-Path $app "SheetalBridge.exe") -Force
$token = [Microsoft.VisualBasic.Interaction]::InputBox("Paste the Agent V token if your server uses one. Leave blank when not required.", "Connect Agent V", "")
$cfg = @{ server=$Server; token=$token.Trim(); device="laptop" } | ConvertTo-Json
Set-Content -Path (Join-Path $app "config.json") -Value $cfg -Encoding UTF8
$action = New-ScheduledTaskAction -Execute (Join-Path $app "SheetalBridge.exe") -WorkingDirectory $app
$trigger = New-ScheduledTaskTrigger -AtLogOn
Register-ScheduledTask -TaskName "Agent V" -Action $action -Trigger $trigger -Description "Connect this Windows laptop to Agent V" -Force | Out-Null
[System.Windows.Forms.MessageBox]::Show("Agent V is installed and will start automatically when you sign in.", "Agent V", "OK", "Information") | Out-Null
