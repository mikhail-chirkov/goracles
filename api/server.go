package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// error struct for upstream propagation
type handlerError struct {
	StatusCode int
	Error      error
	Message    string
}

// Error handling middle for handlers
type handlerMiddleware func(w http.ResponseWriter, r *http.Request) *handlerError

func (f handlerMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := f(w, r)
	if err != nil {
		http.Error(w, err.Message, err.StatusCode)
		// TODO: only internal?
		log.Printf("Server error: %v", err.Error)
	}
}

type server struct {
	weatherClient *weatherClient
}

func newServer(client *weatherClient) *server {
	return &server{weatherClient: client}
}

type predictParameters struct {
	Latitude   float64
	Longtitude float64
	Radius     int
	StartTime  time.Time
	Timezone   string
	TimeFrame  time.Duration
	WindowSize int
}

// TODO: validator
func parsePredictRequest(query *url.Values) (predictParameters, *handlerError) {
	latitude, err := strconv.ParseFloat(query.Get("latitude"), 64)
	if err != nil {
		return predictParameters{}, &handlerError{StatusCode: http.StatusBadRequest, Error: err, Message: "Latitude is invalid"}
	}

	longtitude, err := strconv.ParseFloat(query.Get("longtitude"), 64)
	if err != nil {
		return predictParameters{}, &handlerError{StatusCode: http.StatusBadRequest, Error: err, Message: "Longtitude is invalid"}
	}

	radius, err := strconv.Atoi(query.Get("radius"))
	if err != nil {
		return predictParameters{}, &handlerError{StatusCode: http.StatusBadRequest, Error: err, Message: "Radius is invalid"}
	}
	radius = max(radius, 1000)

	startTime, err := time.Parse(time.RFC3339, query.Get("startTime"))
	if err != nil {
		return predictParameters{}, &handlerError{StatusCode: http.StatusBadRequest, Error: err, Message: "Start time is invalid"}
	}

	timezone := query.Get("timezone")
	if timezone == "" {
		return predictParameters{}, &handlerError{StatusCode: http.StatusBadRequest, Error: fmt.Errorf("missing timezone"), Message: "Timezone is invalid"}
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return predictParameters{}, &handlerError{StatusCode: http.StatusBadRequest, Error: err, Message: "Timezone is invalid"}
	}

	timeFrameRaw, err := strconv.ParseInt(query.Get("timeFrame"), 10, 64)
	if err != nil {
		return predictParameters{}, &handlerError{StatusCode: http.StatusBadRequest, Error: err, Message: "The time frame is invalid"}
	}
	// round the minimal time to 5 mins (lowest interval)
	timeFrame := max(time.Duration(timeFrameRaw)*time.Minute, time.Minute*5)

	tripDurationRaw, err := strconv.ParseInt(query.Get("tripDuration"), 10, 64)
	if err != nil {
		return predictParameters{}, &handlerError{StatusCode: http.StatusBadRequest, Error: err, Message: "The trip duration is invalid"}
	}
	tripDuration := time.Duration(tripDurationRaw) * time.Minute

	if timeFrame < tripDuration {
		return predictParameters{}, &handlerError{StatusCode: http.StatusBadRequest, Error: err, Message: "The trip duration must be shorter or equal to the time frame"}
	}

	windowSize := tripDurationToIntervals(tripDuration)
	if windowSize <= 0 {
		return predictParameters{}, &handlerError{StatusCode: http.StatusBadRequest, Error: err, Message: "The trip duration must be at least 5 minutes"}
	}

	return predictParameters{Latitude: latitude, Longtitude: longtitude, Radius: radius, StartTime: startTime, Timezone: timezone, TimeFrame: timeFrame, WindowSize: windowSize}, nil
}

func (s *server) predictHandler(writer http.ResponseWriter, reader *http.Request) *handlerError {
	query := reader.URL.Query()

	parameters, handlerErr := parsePredictRequest(&query)
	if handlerErr != nil {
		return handlerErr
	}

	radarData, handlerErr := s.weatherClient.getRadarData(
		parameters.Latitude,
		parameters.Longtitude,
		parameters.Radius,
		parameters.StartTime,
		parameters.StartTime.Add(parameters.TimeFrame),
		parameters.Timezone,
	)
	if handlerErr != nil {
		return handlerErr
	}

	if len(radarData.Records) == 0 {
		return &handlerError{StatusCode: http.StatusBadRequest, Error: fmt.Errorf("No radar data is available for these coordinates/time frame"), Message: "No radar data is available for these coordinates/time frame"}
	}

	var intervals []interval

	for _, record := range radarData.Records {
		recordTime, err := time.Parse(time.RFC3339, record.TimeStamp)
		if err != nil {
			return &handlerError{StatusCode: http.StatusInternalServerError, Error: fmt.Errorf("Failed to parse the radar data: %v", err), Message: "Internal error"}
		}

		interval, err := reduceGridToInterval(record.Precipitation, recordTime)
		if err != nil {
			return &handlerError{StatusCode: http.StatusInternalServerError, Error: fmt.Errorf("Failed to compute intervals: %v", err), Message: "Internal error"}
		}
		intervals = append(intervals, interval)
	}

	windows, err := computeWindowsByMaxIntensity(parameters.WindowSize, intervals)
	if err != nil {
		return &handlerError{StatusCode: http.StatusInternalServerError, Error: fmt.Errorf("Failed to compute windows: %v", err), Message: "Internal error"}
	}

	writer.Header().Add("Content-Type", "application/json")
	// encode the response
	err = json.NewEncoder(writer).Encode(windows)
	if err != nil {
		return &handlerError{StatusCode: http.StatusInternalServerError, Error: fmt.Errorf("Failed to encode windows: %v", err), Message: "Internal error"}
	}

	return nil
}
