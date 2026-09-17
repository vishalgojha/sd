# Agent V for Windows

Build `dist/SheetalBridge.exe`, install Inno Setup, and compile
`SheetalBridge.iss` to produce `SheetalBridgeSetup.exe`. The installer runs a
first-use wizard, stores the token under the user's `%APPDATA%` directory, and
registers a per-user scheduled task so the bridge starts at logon. It does not
require administrator access.
