package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Profile represents json data from Tinder API about token owner
type Profile struct {
	ID             string      `json:"_id"`
	AgeFilterMax   int         `json:"age_filter_max"`
	AgeFilterMin   int         `json:"age_filter_min"`
	BirthDate      time.Time   `json:"birth_date"`
	CreateDate     time.Time   `json:"create_date"`
	Discoverable   bool        `json:"discoverable"`
	DistanceFilter int         `json:"distance_filter"`
	Email          string      `json:"email"`
	Gender         int         `json:"gender"`
	Name           string      `json:"name"`
	Position       Coordinates `json:"pos"`
}

// Coordinates geographic data
type Coordinates struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
}

// PingResponse represents response to change coordinates request
type PingResponse struct {
	Status int         `json:"status"`
	Error  interface{} `json:"error"`
}

// GetProfile allows you to get your own profile data
func (api *TinderAPI) GetProfile() (*Profile, error) {
	const endpoint = "/profile"
	const method = "GET"
	headers := &httpHeaders{}
	(*headers)["Content-type"] = []string{api.ContentType}
	rawResponse, err := api.doAPICall(endpoint, method, headers, nil)
	if err != nil {
		return nil, err
	}
	var profile Profile
	if err := json.Unmarshal(rawResponse, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

// ChangeProfilePosition changes token owner position
func (api *TinderAPI) ChangeProfilePosition(newCoordinates Coordinates) error {
	const endpoint = "/user/ping"
	const method = "POST"
	headers := &httpHeaders{}
	(*headers)["Content-type"] = []string{api.ContentType}
	data, err := json.Marshal(newCoordinates)
	if err != nil {
		return fmt.Errorf("Bad coordinates: %v", err)
	}
	rawResponse, err := api.doAPICall(endpoint, method, headers, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("Unable to change coordinates: %v", err)
	}
	var response PingResponse
	if err := json.Unmarshal(rawResponse, &response); err != nil {
		return fmt.Errorf("Bad response from API: %v", err)
	}
	if err, ok := response.Error.(string); ok {
		return fmt.Errorf("Bad API response: %s", err)
	}
	if response.Status != http.StatusOK {
		return fmt.Errorf("Bad API response HTTP status (no error content): %d", response.Status)
	}
	return nil
}
