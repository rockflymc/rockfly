package ts

import (
	"os"
	"path/filepath"
)

// DTSContent contains the complete TypeScript declarations for the Rockfly plugin API.
const DTSContent = `/**
 * Rockfly Minecraft: Bedrock Edition Server Plugin API
 * Auto-generated type definitions for VS Code IntelliSense and compile-time type safety.
 */

declare namespace rockfly {
    /**
     * Listener execution priority levels.
     */
    enum Priority {
        Lowest = 0,
        Low = 1,
        Normal = 2,
        High = 3,
        Highest = 4,
        Monitor = 5
    }

    /**
     * Vector 3D coordinates.
     */
    interface Vec3 {
        x: number;
        y: number;
        z: number;
    }

    /**
     * Online Player interface.
     */
    interface Player {
        name: string;
        uuid: string;
        xuid: string;
        position: Vec3;
        health: number;
        maxHealth: number;
        gameMode: string;
        sendMessage(msg: string): void;
        teleport(pos: Vec3): void;
        setGameMode(mode: string): void;
        heal(amount: number): void;
    }

    /**
     * World interface.
     */
    interface World {
        name: string;
        time: number;
        setTime(time: number): void;
        setBlock(x: number, y: number, z: number, blockName: string): void;
    }

    /**
     * Base cancellable event.
     */
    interface CancellableEvent {
        cancelled: boolean;
        cancel(): void;
    }

    /**
     * Event fired when a player breaks a block.
     */
    interface BlockBreakEvent extends CancellableEvent {
        player: Player;
        x: number;
        y: number;
        z: number;
        blockName: string;
    }

    /**
     * Event fired when a player moves.
     */
    interface PlayerMoveEvent extends CancellableEvent {
        player: Player;
        from: Vec3;
        to: Vec3;
    }

    /**
     * Event fired when a player joins the server.
     */
    interface PlayerJoinEvent {
        player: Player;
    }

    /**
     * Event fired when a player leaves the server.
     */
    interface PlayerQuitEvent {
        player: Player;
    }

    /**
     * Event manager for registering typed listeners.
     */
    namespace events {
        function on<E = any>(
            eventName: string,
            priority: Priority | number,
            handler: (event: E) => void
        ): () => void;
    }

    /**
     * Command manager for registering in-game commands.
     */
    namespace commands {
        function register(
            name: string,
            description: string,
            handler: (player: Player, args: string[]) => void
        ): void;
    }

    /**
     * Log an informative message to the server console.
     */
    function log(...args: any[]): void;
}

// Global convenience bindings
declare const console: {
    log(...args: any[]): void;
    warn(...args: any[]): void;
    error(...args: any[]): void;
};
`

// GenerateDTS writes rockfly.d.ts to the target directory.
func GenerateDTS(pluginsDir string) error {
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return err
	}
	dtsPath := filepath.Join(pluginsDir, "rockfly.d.ts")
	return os.WriteFile(dtsPath, []byte(DTSContent), 0644)
}
