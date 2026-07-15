package scheduler

import (
	"fmt"
	"time"

	"github.com/robfig/cron"
)

// standardParser accepts 5-field cron (minute–weekday) plus @descriptors and @every.
var standardParser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

// nextRunTime returns the next fire time for schedule after from.
// Supported forms (via robfig/cron):
//   - standard 5-field: "0 0 * * *" (minute hour day month weekday)
//   - descriptors: @yearly, @monthly, @weekly, @daily, @hourly
//   - intervals: @every 1h30m
//
// "once" is reserved for ScheduleOnce and returns the zero time.
func nextRunTime(schedule string, from time.Time) (time.Time, error) {
	if schedule == "" {
		return time.Time{}, fmt.Errorf("empty schedule")
	}
	if schedule == "once" {
		return time.Time{}, nil
	}

	sched, err := standardParser.Parse(schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron schedule %q: %w", schedule, err)
	}
	next := sched.Next(from)
	if next.IsZero() {
		return time.Time{}, fmt.Errorf("cron schedule %q produced no next run", schedule)
	}
	return next, nil
}
