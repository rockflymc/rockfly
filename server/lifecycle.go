package server

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/df-mc/dragonfly/server/cmd/admin"
	"github.com/df-mc/dragonfly/server/cmd/gameplay"
	"github.com/df-mc/dragonfly/server/console"
	"github.com/df-mc/dragonfly/server/gui"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/pelletier/go-toml"
)

// DefaultServerProperties is the standard Bedrock server.properties content
// auto-generated on first boot for vanilla parity.
const DefaultServerProperties = `# Rockfly Minecraft: Bedrock Edition Server Properties
server-name=Rockfly Server
gamemode=survival
difficulty=easy
allow-cheats=false
max-players=20
online-mode=false
white-list=false
server-port=19132
server-portv6=19133
view-distance=32
tick-distance=4
player-idle-timeout=30
max-threads=8
level-name=world
level-seed=
default-player-permission-level=member
texturepack-required=false
`

// DefaultEULAContent is the standard eula.txt generated on first launch.
const DefaultEULAContent = `# By changing the setting below to TRUE you are indicating your agreement to the Mojang EULA (https://account.mojang.com/documents/minecraft_eula).
eula=true
`

// LifecycleOptions configures directory and file paths for the server bootstrapper.
type LifecycleOptions struct {
	// BaseDir is the root working directory for server files.
	BaseDir string
	// ConfigPath is the path to config.toml relative to BaseDir.
	ConfigPath string
	// PropertiesPath is the path to server.properties relative to BaseDir.
	PropertiesPath string
	// EULAPath is the path to eula.txt relative to BaseDir.
	EULAPath string
	// PluginsDir is the directory for plugin artifacts.
	PluginsDir string
	// WorldsDir is the directory where worlds reside.
	WorldsDir string
	// LogsDir is the directory where log files reside.
	LogsDir string
	// Logger is the structured logger to use.
	Logger *slog.Logger
	// AutoAcceptEULA specifies if the EULA should be automatically accepted in non-interactive mode.
	AutoAcceptEULA bool
	// EnableDashboard specifies whether the embedded web administration dashboard is started.
	EnableDashboard bool
	// DashboardAddr is the TCP address for the web dashboard (default "127.0.0.1:8080").
	DashboardAddr string
}

// DefaultLifecycleOptions returns standard production lifecycle options.
func DefaultLifecycleOptions() LifecycleOptions {
	return LifecycleOptions{
		BaseDir:         ".",
		ConfigPath:      "config.toml",
		PropertiesPath:  "server.properties",
		EULAPath:        "eula.txt",
		PluginsDir:      "plugins",
		WorldsDir:       "world",
		LogsDir:         "logs",
		Logger:          slog.Default(),
		AutoAcceptEULA:  true,
		EnableDashboard: true,
		DashboardAddr:   "127.0.0.1:8080",
	}
}

// EnsureDirectories ensures that all required server directories exist.
func EnsureDirectories(opts LifecycleOptions) error {
	dirs := []string{
		filepath.Join(opts.BaseDir, opts.PluginsDir),
		filepath.Join(opts.BaseDir, opts.WorldsDir),
		filepath.Join(opts.BaseDir, opts.LogsDir),
		filepath.Join(opts.BaseDir, "config"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}
	return nil
}

// EnsureEULA checks for eula.txt, creating it and prompting or auto-accepting.
func EnsureEULA(opts LifecycleOptions) (bool, error) {
	eulaFile := filepath.Join(opts.BaseDir, opts.EULAPath)
	if data, err := os.ReadFile(eulaFile); err == nil {
		content := strings.ToLower(string(data))
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "eula=") {
				return strings.TrimPrefix(line, "eula=") == "true", nil
			}
		}
	}

	if opts.AutoAcceptEULA {
		if err := os.WriteFile(eulaFile, []byte(DefaultEULAContent), 0644); err != nil {
			return false, fmt.Errorf("write eula.txt: %w", err)
		}
		if opts.Logger != nil {
			opts.Logger.Info("Mojang EULA auto-accepted. Set eula=false in eula.txt to revoke agreement.")
		}
		return true, nil
	}

	return false, nil
}

// EnsureServerProperties checks if server.properties exists, and creates it if missing.
func EnsureServerProperties(opts LifecycleOptions) error {
	propFile := filepath.Join(opts.BaseDir, opts.PropertiesPath)
	if _, err := os.Stat(propFile); os.IsNotExist(err) {
		if err := os.WriteFile(propFile, []byte(DefaultServerProperties), 0644); err != nil {
			return fmt.Errorf("create server.properties: %w", err)
		}
		if opts.Logger != nil {
			opts.Logger.Info("Created default server.properties with standard vanilla settings.")
		}
	}

	if data, err := os.ReadFile(propFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "white-list=") {
				val := strings.TrimPrefix(line, "white-list=")
				if strings.EqualFold(val, "true") {
					player.SetWhitelistEnabled(true)
				}
			}
		}
	}
	return nil
}

// ReadOrCreateConfig loads the config.toml file or initializes it with production defaults.
func ReadOrCreateConfig(opts LifecycleOptions) (Config, error) {
	var zero Config
	configFile := filepath.Join(opts.BaseDir, opts.ConfigPath)

	c := DefaultConfig()
	// Set default server name and offline mode by default
	c.Server.Name = "Rockfly Server"
	c.Server.AuthEnabled = false // Offline mode by default for zero-friction setup

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		data, err := toml.Marshal(c)
		if err != nil {
			return zero, fmt.Errorf("encode default config: %w", err)
		}
		if err := os.WriteFile(configFile, data, 0644); err != nil {
			return zero, fmt.Errorf("create default config: %w", err)
		}
		if opts.Logger != nil {
			opts.Logger.Info("Created default config.toml (offline mode enabled by default).")
		}
		return c.Config(opts.Logger)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return zero, fmt.Errorf("read config: %w", err)
	}
	if err := toml.Unmarshal(data, &c); err != nil {
		return zero, fmt.Errorf("decode config: %w", err)
	}
	return c.Config(opts.Logger)
}

// Bootstrap bootstraps the entire server runtime: ensures directory hierarchy,
// checks EULA, generates default configurations, and instantiates the Server.
func Bootstrap(opts ...LifecycleOptions) (*Server, error) {
	opt := DefaultLifecycleOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.Logger == nil {
		opt.Logger = slog.Default()
	}

	if err := EnsureDirectories(opt); err != nil {
		return nil, fmt.Errorf("bootstrap directories: %w", err)
	}

	agreed, err := EnsureEULA(opt)
	if err != nil {
		return nil, fmt.Errorf("bootstrap eula: %w", err)
	}
	if !agreed {
		return nil, fmt.Errorf("you must accept the Mojang EULA in eula.txt (set eula=true) to run the server")
	}

	if err := EnsureServerProperties(opt); err != nil {
		return nil, fmt.Errorf("bootstrap server.properties: %w", err)
	}

	conf, err := ReadOrCreateConfig(opt)
	if err != nil {
		return nil, fmt.Errorf("bootstrap config: %w", err)
	}

	srv := conf.New()
	srv.SetupGracefulShutdown(opt.Logger)
	admin.RegisterAll(srv)
	gameplay.RegisterAll(srv)
	return srv, nil
}

// SetupGracefulShutdown configures OS interrupt signal handling for clean teardown.
func (srv *Server) SetupGracefulShutdown(log *slog.Logger) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		sig := <-c
		if log != nil {
			log.Info("Shutdown signal received, closing server gracefully...", "signal", sig.String())
		}
		shutdownStart := time.Now()
		if err := srv.Close(); err != nil {
			if log != nil {
				log.Error("error during server shutdown", "error", err)
			}
		}
		if log != nil {
			log.Info("Server stopped cleanly.", "duration", time.Since(shutdownStart).String())
		}
	}()
}

// Run executes the complete turnkey server: bootstraps, starts listeners,
// launches the interactive console REPL, accepts incoming players in a loop,
// and cleanly terminates on exit.
func Run(opts ...LifecycleOptions) error {
	opt := DefaultLifecycleOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Setup colorful console logger
	consoleHandler := console.NewHandler(console.HandlerOptions{
		Level:   slog.LevelInfo,
		LogsDir: opt.LogsDir,
	})
	defer consoleHandler.Close()
	logger := slog.New(consoleHandler)
	slog.SetDefault(logger)
	opt.Logger = logger

	srv, err := Bootstrap(opt)
	if err != nil {
		return err
	}

	repl := console.NewREPL(os.Stdin, logger, ".rockfly_history")
	repl.Start(srv)
	defer repl.Close()

	if opt.EnableDashboard {
		dash := gui.NewDashboardServer(srv, gui.DashboardOptions{
			Addr: opt.DashboardAddr,
		})
		if err := dash.Start(); err != nil {
			logger.Warn("Failed to start web dashboard", "error", err)
		} else {
			logger.Info("Web Administration Dashboard listening", "url", "http://"+dash.Addr())
			defer dash.Close()
		}
	}

	srv.Listen()
	for p := range srv.Accept() {
		_ = p
	}
	return nil
}

// PromptEULAInteractive prompts the user interactively on standard input to accept EULA.
func PromptEULAInteractive(reader *bufio.Reader) bool {
	fmt.Print("Do you agree to the Minecraft EULA (https://account.mojang.com/documents/minecraft_eula)? [y/n]: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}
