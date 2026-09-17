param([string]$ExePath = "$PSScriptRoot\SheetalBridge.exe", [string]$Server = "https://sd.vishalojha.me")
$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName Microsoft.VisualBasic
$app = Join-Path $env:APPDATA "SheetalBridge"
New-Item -ItemType Directory -Force -Path $app | Out-Null
Copy-Item $ExePath (Join-Path $app "SheetalBridge.exe") -Force
$token = [Microsoft.VisualBasic.Interaction]::InputBox("Paste the Sheetal bridge token from the Sheetal admin setup page.", "Connect Sheetal", "")
if ([string]::IsNullOrWhiteSpace($token)) { throw "A bridge token is required." }
$cfg = @{ server=$Server; token=$token.Trim(); device="laptop" } | ConvertTo-Json
Set-Content -Path (Join-Path $app "config.json") -Value $cfg -Encoding UTF8
$action = New-ScheduledTaskAction -Execute (Join-Path $app "SheetalBridge.exe") -WorkingDirectory $app
$trigger = New-ScheduledTaskTrigger -AtLogOn
Register-ScheduledTask -TaskName "Sheetal Bridge" -Action $action -Trigger $trigger -Description "Connect this Windows laptop to Sheetal" -Force | Out-Null
[System.Windows.Forms.MessageBox]::Show("Sheetal Bridge is installed and will start automatically when you sign in.", "Sheetal Bridge", "OK", "Information") | Out-Null
