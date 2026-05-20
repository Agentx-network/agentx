// Package buildinfo is the single source of truth for the AgentX release
// version. Both the desktop sidebar and the agent's system prompt read Version
// from here, so the number the agent reports always matches what the app shows.
//
// Do not edit Version by hand — run `make bump-version VERSION=x.y.z`, which
// updates this constant alongside the Windows VERSIONINFO and NSIS installer.
package buildinfo

// Version is the current AgentX release version.
const Version = "0.8.37"
