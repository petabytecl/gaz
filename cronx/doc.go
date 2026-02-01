// Package cronx provides internal cron expression parsing and schedule calculation.
//
// This package replaces robfig/cron/v3 with a minimal API surface focused on
// the functionality needed by the gaz framework. It provides:
//
//   - Standard 5-field cron expression parsing (minute hour dom month dow)
//   - Descriptor shortcuts (@daily, @hourly, @weekly, @monthly, @yearly, @annually, @every)
//   - CRON_TZ= and TZ= prefix support for timezone-specific schedules
//   - Correct DST transition handling (spring forward skips, fall back runs once)
//   - ConstantDelaySchedule for @every duration expressions
//
// The parser supports the standard cron syntax with named months (jan-dec) and
// days of week (sun-sat), ranges, steps, and wildcards.
//
// Example usage:
//
//	// Parse a standard 5-field expression
//	sched, err := cronx.ParseStandard("0 9 * * mon-fri")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	next := sched.Next(time.Now())
//
//	// Parse with timezone
//	sched, err = cronx.ParseStandard("CRON_TZ=America/New_York 0 9 * * *")
//
//	// Use descriptors
//	sched, err = cronx.ParseStandard("@daily")
//	sched, err = cronx.ParseStandard("@every 5m")
package cronx
