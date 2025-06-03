package main

import (
	"context"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"

	"gopkg.in/ini.v1"
)

func getApps(ctx context.Context) ([]string, error) {
	cmd := exec.Command("flatpak", "list", "--app")
	cmd.Dir = "/var/lib/flatpak"

	log.DebugContext(ctx, "Running command", "command", cmd.Path+" "+strings.Join(cmd.Args, " "))
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
			log.WarnContext(ctx, "Skipping line", "line", app)
			continue
		}
		appID := strings.TrimSpace(data[1])
		if appID == "" {
			continue
		}
		log.DebugContext(ctx, "AppID found", "appid", appID)
		apps = append(apps, appID)
	}
	return apps, nil
}

func getFlatpakApp(ctx context.Context, appID string) (Flatpak, error) {
	cmd := exec.Command("flatpak", "info", "-m", appID)

	log.DebugContext(ctx, "Running command", "command", cmd.Path+" "+strings.Join(cmd.Args, " "), "appid", appID)
	out, err := cmd.Output()
	if err != nil {
		return Flatpak{}, err
	}

	log.DebugContext(ctx, "Parsing toml", "data", string(out))
	var flatpak Flatpak
	cfg, err := ini.Load(out)
	if err != nil {
		return Flatpak{}, err
	}
	err = cfg.MapTo(&flatpak)
	if err != nil {
		return Flatpak{}, err
	}

	flatpak.Application.Command, err = getCommand(ctx, &flatpak)
	if err != nil {
		return Flatpak{}, err
	}

	return flatpak, nil
}

func removeDuplicates(ctx context.Context, apps []Flatpak) []Flatpak {
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
			log.WarnContext(ctx, "Found duplicates", "command", app.Application.Command, "apps", apps)
		}
		return ok
	})
}

func getAllFlatpakApps(ctx context.Context) []Flatpak {
	appsID, err := getApps(ctx)
	if err != nil {
		log.ErrorContext(ctx, "Error getting apps", "err", err)
		os.Exit(1)
	}

	apps := make([]Flatpak, len(appsID))
	wg := sync.WaitGroup{}
	for i, appID := range appsID {
		wg.Add(1)
		go func(i int, appID string) {
			defer wg.Done()
			app, err := getFlatpakApp(ctx, appID)
			if err != nil {
				log.ErrorContext(ctx, "Error getting app", "err", err, "appid", appID)
			}
			// No need to lock since we are writing to different indexes
			apps[i] = app
		}(i, appID)
	}
	wg.Wait()

	apps = removeDuplicates(ctx, apps)

	return apps
}
