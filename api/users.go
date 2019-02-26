package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type userResponse struct {
	Status  int  `json:"status"`
	Results User `json:"results"`
}

type usersResponse struct {
	Status  int    `json:"status"`
	Results []User `json:"results"`
}

// User data API representation
type User struct {
	ID            string    `json:"_id"`
	Name          string    `json:"name"`
	Gender        int       `json:"gender"`
	Bio           string    `json:"bio"`
	BirthDate     time.Time `json:"birth_date"`
	BirthDateInfo string    `json:"birth_date_info"`
	PingTime      time.Time `json:"ping_time"`
	Distance      int       `json:"distance_mi"`
	Jobs          []struct {
		Title struct {
			Displayed bool   `json:"displayed"`
			Name      string `json:"name"`
			ID        string `json:"id"`
		} `json:"title"`
		Company struct {
			Displayed bool   `json:"displayed"`
			Name      string `json:"name"`
			ID        string `json:"id"`
		} `json:"company"`
	} `json:"jobs"`
	Schools []struct {
		Name string `json:"name"`
	} `json:"schools"`
	Photos       []Photo `json:"photos"`
	SNumber      int     `json:"s_number"`
	IsTraveling  bool    `json:"is_traveling"`
	HideAge      bool    `json:"hide_age"`
	HideDistance bool    `json:"hide_distance"`
}

// Photo data API respresentation
type Photo struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	FileName  string `json:"fileName"`
	Extension string `json:"extension"`
}

// GetUsers allows you to get Tinder users around you
func (api *TinderAPI) GetUsers() ([]User, error) {
	const endpoint = "/user/recs"
	const method = "GET"
	headers := &httpHeaders{}
	(*headers)["Content-type"] = []string{api.ContentType}
	rawResponse, err := api.doAPICall(endpoint, method, headers, nil)
	if err != nil {
		return nil, err
	}
	var response usersResponse
	if err := json.Unmarshal(rawResponse, &response); err != nil {
		return nil, err
	}
	if response.Status != http.StatusOK {
		return nil, fmt.Errorf("Got bad API response with status %d", response.Status)
	}
	return response.Results, nil
}

// GetUser get information about specific user
func (api *TinderAPI) GetUser(id string) (*User, error) {
	const method = "GET"
	endpoint := fmt.Sprintf("/user/%s", id)
	headers := &httpHeaders{}
	(*headers)["Content-type"] = []string{api.ContentType}
	rawResponse, err := api.doAPICall(endpoint, method, headers, nil)
	if err != nil {
		return nil, err
	}
	var response userResponse
	if err := json.Unmarshal(rawResponse, &response); err != nil {
		return nil, err
	}
	if response.Status != http.StatusOK {
		return nil, fmt.Errorf("got bad API response with status %d", response.Status)
	}
	return &response.Results, nil

}
