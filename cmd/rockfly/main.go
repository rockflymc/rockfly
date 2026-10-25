package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/player/chat"
)

var (
	version = "1.0.0"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Printf("Rockfly Server version %s\n", version)
			return
		}
	}

	slog.SetLogLoggerLevel(slog.LevelInfo)
	chat.Global.Subscribe(chat.StdoutSubscriber{})

	slog.Info("Starting Rockfly Server", "version", version)

	if err := server.Run(); err != nil {
		slog.Error("fatal server error", "error", err)
		os.Exit(1)
	}
}
