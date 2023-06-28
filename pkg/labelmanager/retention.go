package labelmanager

import "time"

type (
	// RetentionConfig specifies strftime formats and their
	// respective retention periods. Ths is used to generate
	// labels for entries and therefore define how long the
	// entry is retained on disk
	RetentionConfig map[string]time.Duration
)

const (
	durOneDay       = 24 * time.Hour
	durOneMonth     = 31 * durOneDay
	durOneWeek      = 7 * durOneDay
	durTwelveMonths = 12 * durOneMonth
)

// DefaultRetentionConfig defines a two-year retention schema with
// 24 hourly, 7 daily, 4 weekly and 12 monthly backups. Other
// backups are held one hour.
var DefaultRetentionConfig = map[string]time.Duration{
	"%Y-%m":             durTwelveMonths, // Created once per month, first backup of the month
	"%Y-w%V":            durOneMonth,     // Created once per week, first backup of the week
	"%Y-%m-%d":          durOneWeek,      // Created once per day, first backup of the day
	"%Y-%m-%dT%H":       durOneDay,       // Created once per hour, first backup of the hour
	"%Y-%m-%dT%H-%M-%S": time.Hour,       // Created once per second, should hold all backups
}
