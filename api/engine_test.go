package main

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReduceGridToInterval(t *testing.T) {
	valueInMmPerHour := 2 * 12

	grid := [][]int{{valueInMmPerHour, valueInMmPerHour}, {valueInMmPerHour, valueInMmPerHour}}
	timeNow := time.Now()

	interval, err := reduceGridToInterval(grid, timeNow)
	if err != nil {
		assert.Error(t, err)
	}

	expectedIntensity := 2.0 * 4
	assert.InDelta(t, expectedIntensity, interval.Intensity, 0.01)

	expectedSquaredIntensity := math.Pow(2.0, 2) * 4
	assert.InDelta(t, expectedSquaredIntensity, interval.SquaredIntensity, 0.01)

	expectedMaxIntensity := 2.0
	assert.Equal(t, expectedMaxIntensity, interval.MaxIntensity)

	assert.Equal(t, timeNow, interval.Time)
}

func TestDurationToIntervals(t *testing.T) {
	assert.Equal(t, 0, tripDurationToIntervals(time.Minute*0))
	assert.Equal(t, 1, tripDurationToIntervals(time.Minute*4))
	assert.Equal(t, 1, tripDurationToIntervals(time.Minute*5))
	assert.Equal(t, 2, tripDurationToIntervals(time.Minute*6))
}

func TestComputeWindowsByMaxIntensity(t *testing.T) {
	windowSize := 2
	timeNow := time.Now()
	intervals := []interval{interval{MaxIntensity: 5, Time: timeNow}, interval{MaxIntensity: 1, Time: timeNow.Add(time.Minute * 5)}, interval{MaxIntensity: 3, Time: timeNow.Add(time.Minute * 10)}}

	windows, err := computeWindowsByMaxIntensity(windowSize, intervals)
	if err != nil {
		assert.Error(t, err)
	}

	// should output 2 windows
	assert.Equal(t, 2, len(windows))

	first := windows[0]
	second := windows[1]

	// check ordered ascended
	assert.InDelta(t, 2, first.Precipitation, 0.01)
	assert.Equal(t, timeNow.Add(time.Minute*5), first.StartTime)
	assert.Equal(t, first.Description, "Light")

	assert.InDelta(t, 3, second.Precipitation, 0.01)
	assert.Equal(t, timeNow, second.StartTime)
	assert.Equal(t, second.Description, "Moderate")
}

// TODO: test empty
