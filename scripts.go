package main

import (
	"os"
	"path"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"
)

func removeOldScripts(config Config) {
	files, err := os.ReadDir(config.FlatpakDir)
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading output directory")
	}

	for _, file := range files {
		f, err := os.Open(path.Join(config.FlatpakDir, file.Name()))
		if err != nil {
			log.Error().Err(err).Str("file", file.Name()).Msg("Error opening file")
			continue
		}
		content := make([]byte, 30)
		i, err := f.ReadAt(content, 14)
		if err != nil || i != 30 || string(content) != "**Generated by flatpak-alias**" {
			log.Warn().Str("file", file.Name()).Msg("Not a managed file")
			continue
		}

		err = os.Remove(path.Join(config.FlatpakDir, file.Name()))
		if err != nil {
			log.Error().Err(err).Str("file", file.Name()).Msg("Error deleting file")
		}
	}
}

func generateScripts(apps []Flatpak, scriptTemplate *template.Template, config Config) {
	now := time.Now()
	for _, app := range apps {
		log.Info().Str("name", app.Application.Name).Str("command", app.Application.Command).Msg("Found application")
		app.Timestamp = now

		if _, err := os.Stat(path.Join(config.FlatpakDir, app.Application.Command)); err == nil {
			log.Warn().Str("command", app.Application.Command).Msg("File already exists")
			continue
		}

		file, err := os.Create(path.Join(config.FlatpakDir, app.Application.Command))
		if err != nil {
			log.Error().Err(err).Str("command", app.Application.Command).Msg("Error creating file")
			continue
		}
		err = scriptTemplate.Execute(file, app)
		if err != nil {
			log.Error().Err(err).Str("command", app.Application.Command).Msg("Error writing template")
			continue
		}

		os.Chmod(path.Join(config.FlatpakDir, app.Application.Command), 0755)
	}
}