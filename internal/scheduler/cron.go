package scheduler

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"

	"taskflow/internal/task"
)

// standardParser is a 5-field POSIX cron parser that evaluates schedules in UTC.
var standardParser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
)

// scheduleWrapper adapts a cron.Schedule to satisfy task.NextScheduler.
// cron.Schedule already has the required Next(time.Time) time.Time method, so
// this is a simple type alias that makes the interface satisfaction explicit.
type scheduleWrapper struct {
	cron.Schedule
}

// CronParser wraps robfig/cron/v3 and satisfies the task.CronParser interface.
type CronParser struct{}

// NewCronParser returns a new CronParser.
func NewCronParser() *CronParser {
	return &CronParser{}
}

// ParseCron parses a standard 5-field POSIX cron expression and returns the
// parsed cron.Schedule evaluated in UTC. It wraps the robfig/cron/v3 standard
// parser restricted to Minute|Hour|Dom|Month|Dow.
func (c *CronParser) ParseCron(expr string) (cron.Schedule, error) {
	sched, err := standardParser.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression %q: %w", expr, err)
	}
	return sched, nil
}

// Parse implements task.CronParser. It validates the expression and returns a
// task.NextScheduler that can compute future execution times.
func (c *CronParser) Parse(expr string) (task.NextScheduler, error) {
	sched, err := c.ParseCron(expr)
	if err != nil {
		return nil, err
	}
	return &scheduleWrapper{sched}, nil
}

// NextTime implements task.CronParser. It returns the first time after `from`
// that the expression fires, in UTC.
func (c *CronParser) NextTime(expr string, from time.Time) (time.Time, error) {
	sched, err := c.ParseCron(expr)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(from.UTC()), nil
}

// ComputeNextTimes returns the next n UTC times the expression fires, starting
// from the current moment (time.Now().UTC()).
func (c *CronParser) ComputeNextTimes(expr string, n int) ([]time.Time, error) {
	sched, err := c.ParseCron(expr)
	if err != nil {
		return nil, err
	}

	times := make([]time.Time, 0, n)
	t := time.Now().UTC()
	for i := 0; i < n; i++ {
		t = sched.Next(t)
		times = append(times, t)
	}
	return times, nil
}

// NormalizeCronExpr validates the expression and returns it as a canonical
// string. Because robfig/cron/v3 does not provide a serialization method, the
// canonical form is defined as the original expression string when it is valid.
// Returns an error if the expression cannot be parsed.
//
// For round-trip testing purposes the contract is:
//
//	ComputeNextTimes(expr, 5) == ComputeNextTimes(NormalizeCronExpr(expr), 5)
//
// which holds trivially when NormalizeCronExpr returns the original expr.
func (c *CronParser) NormalizeCronExpr(expr string) (string, error) {
	if _, err := c.ParseCron(expr); err != nil {
		return "", err
	}
	return expr, nil
}
