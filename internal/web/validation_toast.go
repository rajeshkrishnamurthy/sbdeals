package web

import (
	"sort"
	"strings"
)

func buildValidationToast(errorsByField map[string]string, fieldOrder []string) string {
	if len(errorsByField) == 0 {
		return ""
	}

	for _, field := range fieldOrder {
		if message := strings.TrimSpace(errorsByField[field]); message != "" {
			return "Please fix: " + message
		}
	}

	keys := make([]string, 0, len(errorsByField))
	for key := range errorsByField {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if message := strings.TrimSpace(errorsByField[key]); message != "" {
			return "Please fix: " + message
		}
	}

	return "Please fix the highlighted fields."
}
