package plugin

// Plugin is the unified interface implemented by all Rockfly plugins (TypeScript and WebAssembly).
type Plugin interface {
	// Manifest returns the metadata and dependency declarations for this plugin.
	Manifest() PluginManifest
	// Enable activates the plugin, running its onEnable lifecycle hook and registering listeners.
	Enable() error
	// Disable deactivates the plugin, unsubscribing all listeners and running onDisable.
	Disable() error
}
