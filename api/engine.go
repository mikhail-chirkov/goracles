package main

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"time"
)

type interval struct {
	Intensity        float64
	SquaredIntensity float64
	MaxIntensity     float64
	Time             time.Time
}

func reduceGridToInterval(precipitationGrid [][]int, recordTime time.Time) (interval, error) {
	if len(precipitationGrid) == 0 || len(precipitationGrid[0]) == 0 {
		return interval{}, fmt.Errorf("empty precipitation grid")
	}

	intensity := float64(0)
	squaredIntensity := float64(0)
	maxIntensity := float64(0)

	// TODO: copy
	for _, row := range precipitationGrid {
		for _, cellValue := range row {
			// convert cell intensity from mm/5 min to mm/h
			cellIntensity := float64(cellValue) / 12

			// add to the total intesity
			intensity += cellIntensity

			// add to the squared intensity
			squaredIntensity += math.Pow(cellIntensity, 2)

			maxIntensity = math.Max(maxIntensity, cellIntensity)
		}
	}

	return interval{Intensity: intensity, SquaredIntensity: squaredIntensity, MaxIntensity: maxIntensity, Time: recordTime}, nil
}

func tripDurationToIntervals(tripDuration time.Duration) int {
	intervalDuration := time.Minute * 5
	// do ceiling division
	return int((tripDuration + intervalDuration - 1) / intervalDuration)
}

func meanIntesityToString(intensity float64) string {
	if intensity == 0 {
		return "Dry"
	} else if intensity < 0.5 {
		return "Very light"
	} else if intensity < 2.5 {
		return "Light"
	} else if intensity < 10 {
		return "Moderate"
	} else {
		return "Heavy"
	}
}

type window struct {
	Precipitation float64   `json:"precipitation"`
	Description   string    `json:"description"`
	StartTime     time.Time `json:"startTime"`
}

func computeWindowsByMaxIntensity(intervalsInWindow int, intervals []interval) ([]*window, error) {
	// TODO: comparison
	if intervalsInWindow >= len(intervals) {
		return []*window{}, fmt.Errorf("The duration of trip is longer than the collected radar date")
	}

	var windows []*window
	intensitySum := float64(0)

	// sliding window on intervals
	for i := int(0); i < len(intervals); i++ {
		interval := &intervals[i]

		intensitySum += interval.MaxIntensity

		// check window complete
		if i >= intervalsInWindow-1 {
			// get the beginning of the window
			firstInterval := &intervals[i+1-intervalsInWindow]

			// add new window
			windows = append(windows, &window{
				Precipitation: intensitySum / float64(intervalsInWindow),
				Description:   meanIntesityToString(intensitySum / float64(intervalsInWindow)),
				StartTime:     firstInterval.Time})

			// remove the first interval
			intensitySum -= firstInterval.MaxIntensity
		}
	}

	// sort by the lowest precipitation
	slices.SortFunc(windows, func(a, b *window) int {
		return cmp.Compare(a.Precipitation, b.Precipitation)
	})

	return windows, nil
}
