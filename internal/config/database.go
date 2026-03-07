package config

import (
	"errors"
	"strings"
)

var ErrMissingDatabaseURL = errors.New("DATABASE_URL is required")

func DatabaseURLFromEnv(getenv func(string) string) (string, error) {
	url := strings.TrimSpace(getenv("DATABASE_URL"))
	if url == "" {
		return "", ErrMissingDatabaseURL
	}
	return url, nil
}
