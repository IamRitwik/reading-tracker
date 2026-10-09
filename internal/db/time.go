package db

import "time"

// nowRFC3339 is used by the migration runner.
// Repository code will use its own helper — kept here so db is self-contained.
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
