[Setup]
AppName=Agent V
AppVersion=1.0.0
DefaultDirName={autopf}\Agent V
DefaultGroupName=Agent V
OutputBaseFilename=AgentVSetup
Compression=lzma
SolidCompression=yes
PrivilegesRequired=lowest
WizardStyle=modern

[Files]
Source: "..\..\dist\SheetalBridge.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "Setup-SheetalBridge.ps1"; DestDir: "{app}"; Flags: ignoreversion

[Run]
Filename: "powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File \"{app}\Setup-SheetalBridge.ps1\" -ExePath \"{app}\SheetalBridge.exe\""; Flags: waituntilterminated

[UninstallRun]
Filename: "schtasks.exe"; Parameters: "/Delete /TN \"Agent V\" /F"; Flags: runhidden
