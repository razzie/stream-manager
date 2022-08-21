package beepboop

import (
	"fmt"
	"html/template"
	"time"
)

// TemplateFuncs is beepboop's built in template FuncMap
var TemplateFuncs = template.FuncMap{
	"TimeElapsed": func(then int64) string {
		return TimeElapsed(time.Now(), time.Unix(then, 0))
	},
	"ByteCountSI":  ByteCountSI,
	"ByteCountIEC": ByteCountIEC,
}

// TimeElapsed returns the elapsed time in human readable format (such as "5 days ago")
func TimeElapsed(now time.Time, then time.Time) string {
	text := func(unit string, amount int64) string {
		if amount > 1 {
			unit += "s"
		}
		if now.After(then) {
			return fmt.Sprintf("%d %s ago", amount, unit)
		}
		return fmt.Sprintf("%d %s after", amount, unit)
	}

	seconds := now.Unix() - then.Unix()
	minutes := seconds / 60
	hours := minutes / 60
	days := hours / 24
	weeks := days / 7
	months := days / 30
	years := days / 365

	switch {
	case years > 0:
		return text("year", years)
	case months > 0:
		return text("month", months)
	case weeks > 0:
		return text("week", weeks)
	case days > 0:
		return text("day", days)
	case hours > 0:
		return text("hour", hours)
	case minutes > 0:
		return text("minute", minutes)
	case seconds > 10:
		return text("second", seconds)
	default:
		return "just now"
	}
}

// ByteCountSI ...
func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

// ByteCountIEC ...
func ByteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}
