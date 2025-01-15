package main

import (
	"errors"
	"strings"

	"github.com/hbollon/go-edlib"
	"github.com/rs/zerolog/log"
)

const threshold = 0.75

const algorithm = edlib.Levenshtein

const (
	matchGood = "good"
	matchSame = "same"
	matchNope = "nope"
)

func toCommand(str string) string {
	res := make([]byte, len(str))
	i := 0
	needSpace := false
	for _, c := range str {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			if needSpace {
				res[i] = '-'
				i++
			}
			res[i] = byte(c)
			i++
			needSpace = false
		} else if c >= 'A' && c <= 'Z' {
			res[i] = '-'
			res[i+1] = byte(c - 'A' + 'a')
			i += 2
			needSpace = false
		} else if c == '.' {
			res[i] = '.'
			i++
			needSpace = false
		} else {
			needSpace = true
		}
	}
	res = res[:i]
	return string(res)
}

func getCommand(app *Flatpak) (string, error) {
	var res string
	var match string
	command := strings.ToLower(strings.TrimSpace(app.Application.Command))
	appID := strings.ToLower(strings.TrimSpace(app.Application.Name))
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
	log.Debug().Str("appID", appID).Str("name", name).Str("command", command).Float32("rank", similarity).Str("result", res).Msgf("Match %s", match)
	return toCommand(res), nil
}
