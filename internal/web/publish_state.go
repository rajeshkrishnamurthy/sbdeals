package web

import (
	"fmt"
	"time"
)

func publishStateLabel(isPublished bool) string {
	if isPublished {
		return "Published"
	}
	return "Unpublished"
}

func publishRecencyLabel(isPublished bool, publishedAt, unpublishedAt *time.Time) string {
	if isPublished {
		return daysSinceLabel(publishedAt)
	}
	return daysSinceLabel(unpublishedAt)
}

func daysSinceLabel(ts *time.Time) string {
	if ts == nil {
		return ""
	}
	now := time.Now().UTC()
	days := int(now.Sub(ts.UTC()).Hours() / 24)
	if days < 0 {
		days = 0
	}
	return fmt.Sprintf("(%dd)", days)
}
