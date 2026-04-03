package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// LiveStatusResponse defines the standard structure for train tracking
type LiveStatusResponse struct {
	CurrentStation string  `json:"current_station"`
	NextStation    string  `json:"next_station"`
	DelayMinutes   int     `json:"delay"`
	Lat            float64 `json:"latitude"`
	Lon            float64 `json:"longitude"`
	LastUpdated    string  `json:"last_updated"`
}

// FetchLiveStatusFromExternalAPI connects to a real-time provider.
// Replace the URL with your preferred provider (e.g., a RailwayAPI or custom scraper).
func FetchLiveStatusFromExternalAPI(trainNumber string) (*LiveStatusResponse, error) {
	apiURL := fmt.Sprintf("https://api.railwaytracking.com/v1/live/%s", trainNumber) // Example Endpoint

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("external api error: status %d", resp.StatusCode)
	}

	var status LiveStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}
