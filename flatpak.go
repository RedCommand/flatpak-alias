package main

import (
	"os/exec"
	"slices"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"gopkg.in/ini.v1"
)

func getApps() ([]string, error) {
	cmd := exec.Command("flatpak", "list", "--app")
	cmd.Dir = "/var/lib/flatpak"

	log.Debug().Str("command", cmd.Path+strings.Join(cmd.Args, " ")).Msg("Running command")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	appsLine := strings.Split(string(out), "\n")
	var apps []string
	for _, app := range appsLine {
		app = strings.TrimSpace(app)
		if app == "" {
			continue
		}
		data := strings.Split(app, "\t")
		if len(data) < 2 {
			log.Warn().Msgf("Skipping line: %s", app)
			continue
		}
		appID := strings.TrimSpace(data[1])
		if appID == "" {
			continue
		}
		log.Trace().Msgf("AppID: %s", appID)
		apps = append(apps, appID)
	}
	return apps, nil
}

func getFlatpakApp(appID string) (Flatpak, error) {
	cmd := exec.Command("flatpak", "info", "-m", appID)

	log.Debug().Str("command", cmd.Path+strings.Join(cmd.Args, " ")).Str("appID", appID).Msg("Running command")
	out, err := cmd.Output()
	if err != nil {
		return Flatpak{}, err
	}

	log.Trace().Str("data", string(out)).Msg("Parsing toml")
	var flatpak Flatpak
	cfg, err := ini.Load(out)
	if err != nil {
		return Flatpak{}, err
	}
	err = cfg.MapTo(&flatpak)
	if err != nil {
		return Flatpak{}, err
	}

	flatpak.Application.Command, err = getCommand(&flatpak)
	if err != nil {
		return Flatpak{}, err
	}

	return flatpak, nil
}

func removeDuplicates(apps []Flatpak) []Flatpak {
	keys := make(map[string]*Flatpak)
	list := []Flatpak{}
	duplicates := make(map[string][]Flatpak)
	for _, app := range apps {
		if f, ok := keys[app.Application.Command]; !ok {
			keys[app.Application.Command] = &app
			list = append(list, app)
		} else {
			if len(duplicates[app.Application.Command]) == 0 {
				duplicates[app.Application.Command] = append(duplicates[app.Application.Command], *f, app)
			} else {
				duplicates[app.Application.Command] = append(duplicates[app.Application.Command], app)
			}
		}
	}
	return slices.DeleteFunc(list, func(app Flatpak) bool {
		apps, ok := duplicates[app.Application.Command]
		if ok {
			log.Warn().Str("command", app.Application.Command).Interface("apps", apps).Msg("Found duplicates")
		}
		return ok
	})
}

func getAllFlatpakApps() []Flatpak {
	appsID, err := getApps()
	if err != nil {
		log.Fatal().Err(err).Msg("Error getting apps")
	}

	apps := make([]Flatpak, len(appsID))
	wg := sync.WaitGroup{}
	for i, appID := range appsID {
		wg.Add(1)
		go func(appID string) {
			defer wg.Done()
			app, err := getFlatpakApp(appID)
			if err != nil {
				log.Error().Err(err).Str("appID", appID).Msg("Error getting app")
			}
			// No need to lock since we are writing to different indexes
			apps[i] = app
		}(appID)
	}
	wg.Wait()

	apps = removeDuplicates(apps)

	return apps
}
