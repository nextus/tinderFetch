package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const (
	tinderAPIHost        = "https://api.gotinder.com"
	tinderAPIContentType = "application/json"
	tinderAPIUserAgent   = "Tinder/7.5.3 (iPhone; iOS 10.3.2; Scale/2.00)"
)

type httpHeaders map[string][]string

// TinderAPI only API methods are exported
type TinderAPI struct {
	token       string
	host        string
	contentType string
	userAgent   string
}

func doHTTPRequest(httpURL *url.URL, method string, headers *httpHeaders, requestBody io.Reader) (*http.Response, error) {
	client := &http.Client{
		Timeout: httpTimeout,
	}
	request, err := http.NewRequest(method, httpURL.String(), requestBody)
	if err != nil {
		return nil, err
	}
	for header, values := range *headers {
		(*request).Header.Set(header, strings.Join(values, ","))
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (api *TinderAPI) doAPICall(endpoint, method string, headers *httpHeaders, requestBody io.Reader) ([]byte, error) {
	apiRequest, err := url.Parse(api.host + endpoint)
	if err != nil {
		return nil, err
	}
	if headers == nil {
		headers = &httpHeaders{}
	}
	(*headers)["User-agent"] = []string{api.userAgent}
	(*headers)["X-Auth-Token"] = []string{api.token}
	response, err := doHTTPRequest(apiRequest, method, headers, requestBody)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		log.Printf("http: got %d status code with content: %v", response.StatusCode, string(body))
		return nil, fmt.Errorf("http: unsuccessful response %d", response.StatusCode)
	}
	return body, nil

}

// Dislike someone by id
func (api *TinderAPI) Dislike(id string) error {
	endpoint := fmt.Sprintf("/pass/%s", id)
	const method = "GET"

	log.Printf("Dislike user: %s", id)
	_, err := api.doAPICall(endpoint, method, nil, nil)
	if err != nil {
		return err
	}
	return nil
}
