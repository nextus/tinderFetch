package main

import (
	"encoding/json"
	"time"
)

type Profile struct {
	ID             string    `json:"_id"`
	AgeFilterMax   int       `json:"age_filter_max"`
	AgeFilterMin   int       `json:"age_filter_min"`
	BirthDate      time.Time `json:"birth_date"`
	CreateDate     time.Time `json:"create_date"`
	Discoverable   bool      `json:"discoverable"`
	DistanceFilter int       `json:"distance_filter"`
	Email          string    `json:"email"`
	Gender         int       `json:"gender"`
	Name           string    `json:"name"`
}

// GetProfile allows you to get your own profile data
func (api *TinderAPI) GetProfile() (*Profile, error) {
	const endpoint = "/profile"
	const method = "GET"
	headers := &httpHeaders{}
	(*headers)["Content-type"] = []string{api.contentType}
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
