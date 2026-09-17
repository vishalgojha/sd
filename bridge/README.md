# Sheetal Bridge

The bridge is a small outbound-polling executable for the owner's laptop. It
supports Linux for development/testing and has a polished Windows installer
under `installer/windows` for the eventual single user. It cross-compiles to
Windows (`GOOS=windows GOARCH=amd64`).
It does not expose a listening port. Run the installer from the Sheetal
website; it asks for `SHEETAL_BRIDGE_TOKEN` only when the server has one
enabled. Everyday actions are
allowed; account changes, messages, purchases, and permission changes should
be held for explicit approval in the Sheetal UI.
