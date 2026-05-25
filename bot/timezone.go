package main

import (
	"fmt"
	"time"

	"github.com/bradfitz/latlong"
)

func timezoneForCoordinates(latitude, longitude float64) (string, *time.Location, error) {
	timezone := latlong.LookupZoneName(latitude, longitude)
	if timezone == "" {
		return "", nil, fmt.Errorf("could not determine timezone for coordinates")
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		return "", nil, fmt.Errorf("could not load timezone %q: %w", timezone, err)
	}

	return timezone, location, nil
}
