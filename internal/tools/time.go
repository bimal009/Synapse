package tools

import (
	"context"
	"time"
)

// CurrentTime returns the current time formatted in the given IANA timezone
// (e.g. "UTC", "America/New_York"). An empty or unknown zone falls back to UTC.
func CurrentTime(_ context.Context, timezone string) (string, error) {
	loc := time.UTC
	if timezone != "" {
		if l, err := time.LoadLocation(timezone); err == nil {
			loc = l
		}
	}
	return time.Now().In(loc).Format(time.RFC3339), nil
}
