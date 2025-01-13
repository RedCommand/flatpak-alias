package main

import (
	_ "embed"
	"os"
	"path"
	"text/template"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
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

func parseArgs() Config {
	config := Config{}
	if len(os.Args) > 1 {
		config.FlatpakDir = os.Args[1]
	} else {
		config.FlatpakDir = "/var/lib/flatpak"
	}
	return config
}

func prepare(config *Config) (*template.Template, *lumberjack.Logger) {
	logFile := &lumberjack.Logger{
		Filename:   path.Join(config.FlatpakDir, "flatpak-alias.log"), // Path to the log file
		MaxSize:    1,                                                 // Maximum size in MB before rotation
		MaxBackups: 3,                                                 // Maximum number of old log files to retain
		MaxAge:     365,                                               // Maximum number of days to retain old log files
		Compress:   true,                                              // Compress rotated files
	}

	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}

	multiWriter := zerolog.MultiLevelWriter(consoleWriter, logFile)

	log.Logger = zerolog.New(multiWriter).With().Timestamp().Logger()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	scriptTemplate, err := template.New("script").Parse(scriptTemplateRaw)
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing template")
	}

	config.FlatpakDir = path.Join(config.FlatpakDir, "aliases")
	log.Info().Str("path", config.FlatpakDir).Msg("Writing script")
	err = os.MkdirAll(config.FlatpakDir, 0755)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating directory")
	}

	return scriptTemplate, logFile
}

func main() {
	config := parseArgs()
	scriptTemplate, logFile := prepare(&config)
	defer logFile.Close()
	log.Info().Msg("Starting flatpak-alias")
	log.Info().Str("path", config.FlatpakDir).Any("args", os.Args).Msg("Writing script")
	apps := getAllFlatpakApps()
	removeOldScripts(config)
	generateScripts(apps, scriptTemplate, config)
}
