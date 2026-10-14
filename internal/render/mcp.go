package render

import (
	"encoding/json"
	"regexp"
	"strings"
)

// MCPServer is one MCP server in OpenCode's shape (the neutral catalog format).
type MCPServer struct {
	Type        string            `json:"type"`
	Command     []string          `json:"command,omitempty"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

// Catalog groups MCP servers: "core" is always offered, other groups are optional.
type Catalog map[string]map[string]MCPServer

// ParseCatalog parses mcp.json after placeholders are filled.
func ParseCatalog(data []byte) (Catalog, error) {
	var c Catalog
	err := json.Unmarshal(data, &c)
	return c, err
}

var envRef = regexp.MustCompile(`\{env:([A-Za-z_][A-Za-z0-9_]*)\}`)

func claudeEnv(s string) string { return envRef.ReplaceAllString(s, "$${$1}") }

// OpenCodeEntry returns the server as an OpenCode config entry. Legacy configs
// keep servers directly under "mcp" and need "enabled"; V2 remote servers with
// header credentials need "oauth": false.
func (s MCPServer) OpenCodeEntry(legacy bool) map[string]any {
	e := map[string]any{"type": s.Type}
	if len(s.Command) > 0 {
		e["command"] = s.Command
	}
	if s.URL != "" {
		e["url"] = s.URL
	}
	if len(s.Headers) > 0 {
		e["headers"] = s.Headers
	}
	if len(s.Environment) > 0 {
		e["environment"] = s.Environment
	}
	if legacy {
		e["enabled"] = true
	} else if s.Type == "remote" {
		e["oauth"] = false
	}
	return e
}

// ClaudeEntry returns the server in Claude Code's `claude mcp add-json` shape.
func (s MCPServer) ClaudeEntry() map[string]any {
	if s.Type == "remote" {
		e := map[string]any{"type": "http", "url": s.URL}
		if len(s.Headers) > 0 {
			h := map[string]string{}
			for k, v := range s.Headers {
				h[k] = claudeEnv(v)
			}
			e["headers"] = h
		}
		return e
	}
	e := map[string]any{"type": "stdio"}
	if len(s.Command) > 0 {
		e["command"] = s.Command[0]
		e["args"] = append([]string{}, s.Command[1:]...)
	}
	if len(s.Environment) > 0 {
		env := map[string]string{}
		for k, v := range s.Environment {
			env[k] = claudeEnv(v)
		}
		e["env"] = env
	}
	return e
}

// ServerNames lists every server name in the catalogs, for MCP tool naming.
func ServerNames(catalogs ...map[string]MCPServer) []string {
	seen := map[string]bool{}
	var names []string
	for _, c := range catalogs {
		for n := range c {
			if !seen[n] {
				seen[n] = true
				names = append(names, n)
			}
		}
	}
	return names
}

// ActionServer returns the server prefix of an MCP permission action, if any.
func ActionServer(action string, servers []string) string {
	best := ""
	for _, s := range servers {
		if strings.HasPrefix(action, s+"_") && len(s) > len(best) {
			best = s
		}
	}
	return best
}
