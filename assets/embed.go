// Package assets embeds the neutral sources the installer renders for each assistant.
package assets

import "embed"

// FS holds agents/, skills/, rules/, plugins/ and mcp.json.
//
//go:embed agents skills rules plugins mcp.json
var FS embed.FS
