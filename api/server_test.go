package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPredictComponent(t *testing.T) {

	timeNow := time.Now()

	// create the mock weather api
	mockBrightskyAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/radar" {
			assert.Error(t, fmt.Errorf("Endpoint %s is not allowed", r.URL.Path))
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(radarResponse{
			Records: []radarRecord{
				{TimeStamp: timeNow.Add(time.Minute * 5).Format(time.RFC3339), Source: "dummy", Precipitation: [][]int{{4, 4}, {4, 4}}},
				{TimeStamp: timeNow.Add(time.Minute * 10).Format(time.RFC3339), Source: "dummy", Precipitation: [][]int{{1, 1}, {1, 1}}},
				{TimeStamp: timeNow.Add(time.Minute * 15).Format(time.RFC3339), Source: "dummy", Precipitation: [][]int{{0, 0}, {0, 0}}},
				{TimeStamp: timeNow.Add(time.Minute * 20).Format(time.RFC3339), Source: "dummy", Precipitation: [][]int{{3, 3}, {3, 3}}},
			},
		})
	}))
	defer mockBrightskyAPI.Close()

	// create the server
	mockURL, _ := url.Parse(mockBrightskyAPI.URL)
	server := newServer(newWeatherClient(mockURL))

	// create the request
	request := httptest.NewRequest("GET", "/predict", nil)
	query := url.Values{}
	query.Add("latitude", "55.55")
	query.Add("longtitude", "44.44")
	query.Add("radius", "10000")
	query.Add("timeFrame", "20")
	query.Add("tripDuration", "10")
	request.URL.RawQuery = query.Encode()

	// execute
	responseWriter := httptest.NewRecorder()
	server.predictHandler(responseWriter, request)

	// verify the result
	result := responseWriter.Result()
	assert.Equal(t, http.StatusOK, result.StatusCode)

	var windows []window
	err := json.NewDecoder(result.Body).Decode(&windows)
	if err != nil {
		assert.Error(t, err)
	}

	// should be 3 windows total
	assert.Equal(t, len(windows), 3)
	// verify the best window
	assert.WithinDuration(t, windows[0].StartTime, timeNow.Add(time.Minute*10), time.Second)
}
