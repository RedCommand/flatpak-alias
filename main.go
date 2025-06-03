package main

import (
	"context"
	_ "embed"
	"os"
	"path"
	"text/template"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
	"log/slog"
)

type Config struct {
	FlatpakDir string
}

type Flatpak struct {
	Application struct {
		Name    string `ini:"name"`
		Command string `ini:"command"`
	} `ini:"Application"`
	Timestamp time.Time `ini:"-"`
}

//go:embed script.gotmpl
var scriptTemplateRaw string

var log *slog.Logger

func parseArgs() Config {
	config := Config{}
	if len(os.Args) > 1 {
		config.FlatpakDir = os.Args[1]
	} else {
		config.FlatpakDir = "/var/lib/flatpak"
	}
	return config
}

func prepare(ctx context.Context, config *Config) (*template.Template, *lumberjack.Logger) {
	logFile := &lumberjack.Logger{
		Filename:   path.Join(config.FlatpakDir, "flatpak-alias.log"), // Path to the log file
		MaxSize:    1,                                                 // Maximum size in MB before rotation
		MaxBackups: 3,                                                 // Maximum number of old log files to retain
		MaxAge:     365,                                               // Maximum number of days to retain old log files
		Compress:   true,                                              // Compress rotated files
	}

	textHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	jsonHandler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{Level: slog.LevelInfo})
	log = slog.New(multiHandler{handlers: []slog.Handler{textHandler, jsonHandler}})
	slog.SetDefault(log)

	scriptTemplate, err := template.New("script").Parse(scriptTemplateRaw)
	if err != nil {
		log.ErrorContext(ctx, "Error parsing template", "err", err)
		os.Exit(1)
	}

	config.FlatpakDir = path.Join(config.FlatpakDir, "aliases")
	log.InfoContext(ctx, "Writing script", "path", config.FlatpakDir)
	err = os.MkdirAll(config.FlatpakDir, 0755)
	if err != nil {
		log.ErrorContext(ctx, "Error creating directory", "err", err)
		os.Exit(1)
	}

	return scriptTemplate, logFile
}

func main() {
	ctx := context.Background()
	config := parseArgs()
	scriptTemplate, logFile := prepare(ctx, &config)
	defer logFile.Close()
	log.InfoContext(ctx, "Starting flatpak-alias")
	log.InfoContext(ctx, "Writing script", "path", config.FlatpakDir, "args", os.Args)
	apps := getAllFlatpakApps(ctx)
	removeOldScripts(ctx, config)
	generateScripts(ctx, apps, scriptTemplate, config)
}
