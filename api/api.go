package api

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultTinderAPIHost        = "https://api.gotinder.com"
	DefaultTinderAPIContentType = "application/json"
	DefaultTinderAPIUserAgent   = "Tinder/7.5.3 (iPhone; iOS 10.3.2; Scale/2.00)"
)

type httpHeaders map[string][]string

// TinderAPI only API methods are exported
type TinderAPI struct {
	token       string
	Host        string
	ContentType string
	UserAgent   string
	Timeout     time.Duration
}

func doHTTPRequest(timeout time.Duration, httpURL *url.URL, method string, headers *httpHeaders, requestBody io.Reader) (*http.Response, error) {
	client := &http.Client{
		Timeout: timeout,
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
	apiRequest, err := url.Parse(api.Host + endpoint)
	if err != nil {
		return nil, err
	}
	if headers == nil {
		headers = &httpHeaders{}
	}
	(*headers)["User-agent"] = []string{api.UserAgent}
	(*headers)["X-Auth-Token"] = []string{api.token}
	response, err := doHTTPRequest(api.Timeout, apiRequest, method, headers, requestBody)
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

// New return new API object
func New(token string, timeout time.Duration) *TinderAPI {
	return &TinderAPI{
		token:       token,
		Host:        DefaultTinderAPIHost,
		ContentType: DefaultTinderAPIContentType,
		UserAgent:   DefaultTinderAPIUserAgent,
		Timeout:     timeout,
	}
}
