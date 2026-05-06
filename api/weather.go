package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type weatherClient struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
}

func newWeatherClient(brightskyURL *url.URL) *weatherClient {
	return &weatherClient{
		BaseURL:    brightskyURL,
		HTTPClient: &http.Client{Timeout: time.Second * 10},
	}
}

type radarRecord struct {
	TimeStamp     string  `json:"timestamp"`
	Source        string  `json:"source"`
	Precipitation [][]int `json:"precipitation_5"`
}

type radarResponse struct {
	Records []radarRecord `json:"radar"`
}

// TODO: time/time zone
func (client *weatherClient) getRadarData(latitude float64, longtitude float64, distance int, endTime time.Time) (radarResponse, *handlerError) {
	// construct the endpoint url
	requestUrl := client.BaseURL.JoinPath("radar")

	// add query parameters
	queryParameters := url.Values{}
	queryParameters.Add("distance", strconv.Itoa(distance))
	queryParameters.Add("lat", strconv.FormatFloat(latitude, 'f', -1, 64))
	queryParameters.Add("lon", strconv.FormatFloat(longtitude, 'f', -1, 64))
	queryParameters.Add("date", time.Now().Format(time.RFC3339))
	queryParameters.Add("last_date", endTime.Format(time.RFC3339))
	queryParameters.Add("format", "plain")
	requestUrl.RawQuery = queryParameters.Encode()

	// create the request
	request, err := http.NewRequest("GET", requestUrl.String(), nil)
	if err != nil {
		return radarResponse{}, &handlerError{StatusCode: http.StatusInternalServerError, Error: err, Message: "Internal error"}
	}

	// add headers
	request.Header.Add("Accept", "application/json")

	// do the request
	response, requestError := client.HTTPClient.Do(request)
	if requestError != nil {
		return radarResponse{}, &handlerError{StatusCode: http.StatusBadGateway, Error: err, Message: "Unable to access the Brightsky API"}
	}

	defer response.Body.Close()

	// handle API errors
	if response.StatusCode == 404 || response.StatusCode == 422 {
		var errorDetails map[string]any
		_ = json.NewDecoder(response.Body).Decode(&errorDetails)

		return radarResponse{}, &handlerError{StatusCode: response.StatusCode, Error: fmt.Errorf("Brightsky API error: %s", errorDetails["detail"]), Message: "Input rejected by Brightsky API"}
	}

	// handle other errors
	if response.StatusCode != http.StatusOK {
		return radarResponse{}, &handlerError{StatusCode: response.StatusCode, Error: fmt.Errorf("Unknown Brightsky API error with code %d", response.StatusCode), Message: "Brightsky API error"}
	}

	// parse the response
	var radarData radarResponse
	decodeError := json.NewDecoder(response.Body).Decode(&radarData)
	if decodeError != nil {
		return radarResponse{}, &handlerError{StatusCode: http.StatusInternalServerError, Error: err, Message: "Internal error"}
	}

	// TODO: drop the first

	return radarData, nil
}
