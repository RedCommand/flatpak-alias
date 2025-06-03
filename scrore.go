package main

import (
	"context"
	"errors"
	"strings"

	"github.com/hbollon/go-edlib"
)

const threshold = 0.75

const algorithm = edlib.Levenshtein

const (
	matchGood = "good"
	matchSame = "same"
	matchNope = "nope"
)

func toCommand(ctx context.Context, str string) string {
	res := make([]byte, 0, len(str))
	needSpace := false
	lastUpper := false
	i := 0
	for _, c := range str {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			if needSpace {
				res = append(res, '-')
				i++
			}
			res = append(res, byte(c))
			i++
			needSpace = false
			lastUpper = false
		} else if c >= 'A' && c <= 'Z' {
			if i > 0 && !lastUpper {
				res = append(res, '-')
				i++
			}
			res = append(res, byte(c-'A'+'a'))
			i++
			needSpace = false
			lastUpper = true
		} else if c == '.' {
			res = append(res, '.')
			i++
			needSpace = false
		} else if i > 0 {
			needSpace = true
		}
	}
	log.DebugContext(ctx, "Converted", "input", str, "output", string(res))
	return string(res)
}

func getCommand(ctx context.Context, app *Flatpak) (string, error) {
	var res string
	var match string
	command := strings.TrimSpace(app.Application.Command)
	appID := strings.TrimSpace(app.Application.Name)
	name := appID[strings.LastIndex(appID, ".")+1:]

	similarity, err := edlib.StringsSimilarity(command, name, algorithm)
	if err != nil {
		return "", err
	}
	if similarity < threshold {
		res = name
		match = matchNope
	} else if similarity >= 1 {
		res = name
		match = matchSame
	} else {
		res = command
		match = matchGood
	}
	if res == "" {
		return "", errors.New("empty command")
	}
	log.DebugContext(ctx, "Match", "appid", appID, "name", name, "command", command, "rank", similarity, "result", res, "match", match)
	return toCommand(ctx, res), nil
}
