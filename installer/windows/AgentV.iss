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
Source: "..\..\dist\AgentV.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "Setup-AgentV.ps1"; DestDir: "{app}"; Flags: ignoreversion

[Run]
Filename: "powershell.exe"; Parameters: "-NoProfile -ExecutionPolicy Bypass -File \"{app}\Setup-AgentV.ps1\" -ExePath \"{app}\AgentV.exe\""; Flags: waituntilterminated

[UninstallRun]
Filename: "schtasks.exe"; Parameters: "/Delete /TN \"Agent V\" /F"; Flags: runhidden
