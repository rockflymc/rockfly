package gui

import "embed"

// Assets embeds the static frontend files for the web administration dashboard.
//
//go:embed assets/*
var Assets embed.FS
