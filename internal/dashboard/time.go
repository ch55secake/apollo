package dashboard

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var relativeTimePattern = regexp.MustCompile(`^now(-|\+)([0-9]+)(ms|s|m|h|d|w)$`)

func (r TimeRange) Resolve(now time.Time) (time.Time, time.Time, error) {
	from := strings.TrimSpace(r.From)
	to := strings.TrimSpace(r.To)
	if from == "" {
		from = "now-6h"
	}
	if to == "" {
		to = "now"
	}

	start, err := parseTime(from, now)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse dashboard start time: %w", err)
	}
	end, err := parseTime(to, now)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parse dashboard end time: %w", err)
	}
	if !start.Before(end) {
		return time.Time{}, time.Time{}, fmt.Errorf("dashboard start time must be before end time")
	}
	return start, end, nil
}

func parseTime(value string, now time.Time) (time.Time, error) {
	if value == "now" {
		return now, nil
	}
	match := relativeTimePattern.FindStringSubmatch(value)
	if len(match) == 4 {
		amount, err := strconv.ParseInt(match[2], 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		duration, err := relativeDuration(amount, match[3])
		if err != nil {
			return time.Time{}, err
		}
		if match[1] == "-" {
			return now.Add(-duration), nil
		}
		return now.Add(duration), nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("unsupported time %q", value)
	}
	return parsed, nil
}

func relativeDuration(amount int64, unit string) (time.Duration, error) {
	switch unit {
	case "ms":
		return time.Duration(amount) * time.Millisecond, nil
	case "s":
		return time.Duration(amount) * time.Second, nil
	case "m":
		return time.Duration(amount) * time.Minute, nil
	case "h":
		return time.Duration(amount) * time.Hour, nil
	case "d":
		return time.Duration(amount) * 24 * time.Hour, nil
	case "w":
		return time.Duration(amount) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported duration unit %q", unit)
	}
}
