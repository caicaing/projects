package taskflow

import (
	"time"

	"git.woa.com/mooyliu/mooy_test/framework/internal/schedule"
)

// NormalizeSchedule converts time selections to a scheduler expression.
func NormalizeSchedule(s Schedule) (Schedule, error) {
	return schedule.Normalize(s)
}

// PreviewSchedule returns the next five times without changing state.
func PreviewSchedule(s Schedule, after time.Time) ([]time.Time, error) {
	return schedule.Preview(s, after)
}
