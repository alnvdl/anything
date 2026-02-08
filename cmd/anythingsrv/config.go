package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/alnvdl/anything/internal/app"
)

const (
	defaultDBPath              = "db.json"
	defaultPersistInterval     = 15 * time.Minute
	defaultHealthCheckInterval = 3 * time.Minute
)

// DBPath reads the DB_PATH environment variable. If not set, it defaults to
// "db.json".
func DBPath() string {
	s := os.Getenv("DB_PATH")
	if s == "" {
		return defaultDBPath
	}
	return s
}

// PersistInterval reads and validates the PERSIST_INTERVAL environment
// variable. If not set, it defaults to 15 minutes.
func PersistInterval() time.Duration {
	s := os.Getenv("PERSIST_INTERVAL")
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return defaultPersistInterval
}

// HealthCheckInterval reads and validates the HEALTH_CHECK_INTERVAL
// environment variable. If not set, it defaults to 3 minutes.
func HealthCheckInterval() time.Duration {
	s := os.Getenv("HEALTH_CHECK_INTERVAL")
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return defaultHealthCheckInterval
}

// Port reads and validates the PORT environment variable.
func Port() (int, error) {
	s := os.Getenv("PORT")
	if s == "" {
		return 0, fmt.Errorf("PORT is not set")
	}
	port, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("PORT is not a valid integer: %w", err)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("PORT must be between 1 and 65535")
	}
	return port, nil
}

// Entries reads and validates the ENTRIES environment variable.
func Entries() ([]app.Entry, error) {
	s := os.Getenv("ENTRIES")
	if s == "" {
		return nil, fmt.Errorf("ENTRIES is not set")
	}
	var entries []app.Entry
	if err := json.Unmarshal([]byte(s), &entries); err != nil {
		return nil, fmt.Errorf("ENTRIES is not valid JSON: %w", err)
	}
	return entries, nil
}

// People reads and validates the PEOPLE environment variable.
func People() (map[string]string, error) {
	s := os.Getenv("PEOPLE")
	if s == "" {
		return nil, fmt.Errorf("PEOPLE is not set")
	}
	var people map[string]string
	if err := json.Unmarshal([]byte(s), &people); err != nil {
		return nil, fmt.Errorf("PEOPLE is not valid JSON: %w", err)
	}
	return people, nil
}

// Timezone reads and validates the TIMEZONE environment variable.
func Timezone() (*time.Location, error) {
	s := os.Getenv("TIMEZONE")
	if s == "" {
		return nil, fmt.Errorf("TIMEZONE is not set")
	}
	loc, err := time.LoadLocation(s)
	if err != nil {
		return nil, fmt.Errorf("TIMEZONE is not valid: %w", err)
	}
	return loc, nil
}

// Periods reads and validates the PERIODS environment variable.
// It validates that period hours do not overlap.
func Periods() (app.Periods, error) {
	s := os.Getenv("PERIODS")
	if s == "" {
		return nil, fmt.Errorf("PERIODS is not set")
	}
	var raw map[string][2]int
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return nil, fmt.Errorf("PERIODS is not valid JSON: %w", err)
	}

	// Validate no overlapping hours.
	seen := make(map[int]string)
	for name, bounds := range raw {
		start, end := bounds[0], bounds[1]
		if start == end {
			return nil, fmt.Errorf("PERIODS: period %q has equal start and end hour", name)
		}
		hours := hoursForPeriod(start, end)
		for _, h := range hours {
			if other, ok := seen[h]; ok {
				return nil, fmt.Errorf("PERIODS: hour %d overlaps between %q and %q", h, other, name)
			}
			seen[h] = name
		}
	}

	return app.Periods(raw), nil
}

// GroupOrder reads and validates the GROUP_ORDER environment variable.
// If not set, it returns nil (no custom ordering).
func GroupOrder() ([]string, error) {
	s := os.Getenv("GROUP_ORDER")
	if s == "" {
		return nil, nil
	}
	var order []string
	if err := json.Unmarshal([]byte(s), &order); err != nil {
		return nil, fmt.Errorf("GROUP_ORDER is not valid JSON: %w", err)
	}
	return order, nil
}

// hoursForPeriod returns the list of hours covered by a period [start, end).
func hoursForPeriod(start, end int) []int {
	var hours []int
	if start < end {
		for h := start; h < end; h++ {
			hours = append(hours, h)
		}
	} else {
		// Wraps around midnight.
		for h := start; h < 24; h++ {
			hours = append(hours, h)
		}
		for h := range end {
			hours = append(hours, h)
		}
	}
	return hours
}
