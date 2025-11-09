package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func ParseDate(value string) (*time.Time, string, error) {
	value = strings.TrimSpace(value)

	if strings.ToLower(value) == "none" {
		return nil, "none", nil
	}

	if t, ok := parseRelativeKeyword(strings.ToLower(value)); ok {
		return &t, "", nil
	}

	if t, err := parseRelativeOffset(value); err == nil {
		return &t, "", nil
	}

	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"02-01-2006",
		"02/01/2006",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return &t, "", nil
		}
	}

	return nil, "", fmt.Errorf("unable to parse date: %s (expected ISO date, relative keyword, or offset)", value)
}

func parseRelativeKeyword(value string) (time.Time, bool) {
	now := time.Now()

	switch value {
	case "today":
		return startOfDay(now), true
	case "tomorrow":
		return startOfDay(now.AddDate(0, 0, 1)), true
	case "yesterday":
		return startOfDay(now.AddDate(0, 0, -1)), true
	default:
		return time.Time{}, false
	}
}

func parseRelativeOffset(value string) (time.Time, error) {
	re := regexp.MustCompile(`^([+-]?)(\d+)([mhdwMy])$`)
	matches := re.FindStringSubmatch(value)

	if matches == nil {
		return time.Time{}, fmt.Errorf("invalid offset format")
	}

	sign := matches[1]
	numStr := matches[2]
	unit := matches[3]

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid number in offset: %s", numStr)
	}

	if sign == "-" {
		num = -num
	}

	now := time.Now()
	var result time.Time

	switch unit {
	case "m":
		result = now.Add(time.Duration(num) * time.Minute)
	case "h":
		result = now.Add(time.Duration(num) * time.Hour)
	case "d":
		result = now.AddDate(0, 0, num)
	case "w":
		result = now.AddDate(0, 0, num*7)
	case "M":
		result = now.AddDate(0, num, 0)
	case "y":
		result = now.AddDate(num, 0, 0)
	default:
		return time.Time{}, fmt.Errorf("unknown unit: %s", unit)
	}

	return startOfDay(result), nil
}

func ParseDateRange(value string, operator string) (*time.Time, *time.Time, error) {
	if operator == "<" || operator == "<=" {
		t, _, err := ParseDate(value)
		if err != nil {
			return nil, nil, err
		}
		if operator == "<" {
			return nil, t, nil
		}
		endOfDay := endOfDay(*t)
		return nil, &endOfDay, nil
	}

	if operator == ">" || operator == ">=" {
		t, _, err := ParseDate(value)
		if err != nil {
			return nil, nil, err
		}
		if operator == ">" {
			endOfDay := endOfDay(*t)
			return &endOfDay, nil, nil
		}
		return t, nil, nil
	}

	if strings.Contains(value, "..") {
		parts := strings.Split(value, "..")
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("invalid range syntax: %s", value)
		}

		start, _, err := ParseDate(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, nil, fmt.Errorf("invalid range start: %w", err)
		}

		end, _, err := ParseDate(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, nil, fmt.Errorf("invalid range end: %w", err)
		}

		endOfDay := endOfDay(*end)
		return start, &endOfDay, nil
	}

	t, special, err := ParseDate(value)
	if err != nil {
		return nil, nil, err
	}

	if special == "none" {
		return nil, nil, nil
	}

	endOfDay := endOfDay(*t)
	return t, &endOfDay, nil
}

func startOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func endOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 23, 59, 59, 999999999, t.Location())
}

func FormatDateForSQL(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func FormatDateForDisplay(t time.Time) string {
	return t.Format("2006-01-02")
}
