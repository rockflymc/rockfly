package main

import (
	"log/slog"
	"os"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/player/chat"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelInfo)
	chat.Global.Subscribe(chat.StdoutSubscriber{})

	if err := server.Run(); err != nil {
		slog.Error("fatal server error", "error", err)
		os.Exit(1)
	}
}
