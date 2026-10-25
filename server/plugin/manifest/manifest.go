package manifest

// PluginManifest defines metadata for Rockfly plugins.
type PluginManifest struct {
	Name         string   `toml:"name" json:"name"`
	Version      string   `toml:"version" json:"version"`
	Author       string   `toml:"author" json:"author"`
	Main         string   `toml:"main" json:"main"` // e.g., "index.ts", "index.js", or "plugin.wasm"
	Dependencies []string `toml:"dependencies" json:"dependencies"`
}
