package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/df-mc/dragonfly/server/plugin/ts"
)

// ScaffoldPlugin generates a starter plugin workspace for TypeScript or Go.
func ScaffoldPlugin(name, lang, targetDir string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("plugin name %q contains invalid filesystem characters", name)
	}

	if targetDir == "" {
		targetDir = name
	}
	targetDir = filepath.Clean(targetDir)

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

	lang = strings.ToLower(strings.TrimSpace(lang))
	switch lang {
	case "ts", "typescript":
		return scaffoldTypeScript(name, targetDir)
	case "go", "golang", "wasm":
		return scaffoldGo(name, targetDir)
	default:
		return fmt.Errorf("unsupported language %q: choose 'ts' or 'go'", lang)
	}
}

func scaffoldTypeScript(name, targetDir string) error {
	manifestToml := fmt.Sprintf(`name = "%s"
version = "1.0.0"
author = "Developer"
main = "index.ts"
dependencies = []
`, name)
	if err := os.WriteFile(filepath.Join(targetDir, "plugin.toml"), []byte(manifestToml), 0644); err != nil {
		return err
	}

	indexTS := fmt.Sprintf(`/**
 * %s - Rockfly TypeScript Plugin
 */

rockfly.log("%s enabled successfully!");

rockfly.events.on("PlayerJoinEvent", rockfly.Priority.Normal, (e) => {
    rockfly.log("Welcome player: " + e.player.name);
});

rockfly.commands.register("greet", "Greets the player", (p, args) => {
    if (p) {
        p.sendMessage("Hello from %s!");
    }
});
`, name, name, name)
	if err := os.WriteFile(filepath.Join(targetDir, "index.ts"), []byte(indexTS), 0644); err != nil {
		return err
	}

	tsconfig := `{
  "compilerOptions": {
    "target": "ES2020",
    "module": "CommonJS",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true
  },
  "include": [
    "**/*.ts",
    "rockfly.d.ts"
  ]
}
`
	if err := os.WriteFile(filepath.Join(targetDir, "tsconfig.json"), []byte(tsconfig), 0644); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(targetDir, "rockfly.d.ts"), []byte(ts.DTSContent), 0644)
}

func scaffoldGo(name, targetDir string) error {
	manifestToml := fmt.Sprintf(`name = "%s"
version = "1.0.0"
author = "Developer"
main = "plugin.wasm"
dependencies = []
`, name)
	if err := os.WriteFile(filepath.Join(targetDir, "plugin.toml"), []byte(manifestToml), 0644); err != nil {
		return err
	}

	mainGo := fmt.Sprintf(`package main

import (
	"fmt"
	"github.com/df-mc/dragonfly/sdk/go/rockfly"
)

func main() {
	rockfly.Log("%s WASM plugin initialized!")
}

//export onEnable
func onEnable() {
	rockfly.Log("%s enabled!")
}

//export onDisable
func onDisable() {
	rockfly.Log("%s disabled!")
}
`, name, name, name)
	if err := os.WriteFile(filepath.Join(targetDir, "main.go"), []byte(mainGo), 0644); err != nil {
		return err
	}

	goMod := fmt.Sprintf(`module %s

go 1.27
`, strings.ToLower(name))
	return os.WriteFile(filepath.Join(targetDir, "go.mod"), []byte(goMod), 0644)
}
