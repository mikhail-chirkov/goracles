package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type predictClient struct {
	baseURL    *url.URL
	httpClient *http.Client
}

type predictWindow struct {
	Precipitation float64   `json:"precipitation"`
	Description   string    `json:"description"`
	StartTime     time.Time `json:"startTime"`
	EndTime       time.Time `json:"endTime"`
}

func newPredictClient(baseURL *url.URL) *predictClient {
	return &predictClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *predictClient) predict(ctx context.Context, latitude, longitude float64, defaults userDefaults, startTime time.Time, timezone string) ([]predictWindow, error) {
	endpoint := c.baseURL.JoinPath("predict")

	query := endpoint.Query()
	query.Set("latitude", strconv.FormatFloat(latitude, 'f', -1, 64))
	query.Set("longtitude", strconv.FormatFloat(longitude, 'f', -1, 64))
	query.Set("radius", strconv.Itoa(int(defaults.DefaultRadiusKM*1000)))
	query.Set("tripDuration", strconv.Itoa(defaults.TripDurationMin))
	query.Set("timeFrame", strconv.Itoa(defaults.TimeFrameMin))
	query.Set("startTime", startTime.Format(time.RFC3339))
	query.Set("timezone", timezone)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		if len(body) == 0 {
			return nil, fmt.Errorf("predict API returned %s", response.Status)
		}

		return nil, fmt.Errorf("predict API returned %s: %s", response.Status, string(body))
	}

	var windows []predictWindow
	if err := json.NewDecoder(response.Body).Decode(&windows); err != nil {
		return nil, err
	}

	return windows, nil
}
