package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the API client for Nebula server
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient() *Client {
	return &Client{
		baseURL: GetServerURL(),
		token:   GetToken(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetToken sets the authentication token
func (c *Client) SetToken(token string) {
	c.token = token
}

// Request makes an HTTP request to the API
func (c *Client) Request(method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return c.httpClient.Do(req)
}

// Get makes a GET request
func (c *Client) Get(path string) (*http.Response, error) {
	return c.Request("GET", path, nil)
}

// Post makes a POST request
func (c *Client) Post(path string, body interface{}) (*http.Response, error) {
	return c.Request("POST", path, body)
}

// Put makes a PUT request
func (c *Client) Put(path string, body interface{}) (*http.Response, error) {
	return c.Request("PUT", path, body)
}

// Delete makes a DELETE request
func (c *Client) Delete(path string) (*http.Response, error) {
	return c.Request("DELETE", path, nil)
}

// ParseResponse parses a JSON response
func ParseResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("request failed with status %d", resp.StatusCode)
		}
		return fmt.Errorf("%s", errResp.Error)
	}

	if v != nil {
		return json.NewDecoder(resp.Body).Decode(v)
	}
	return nil
}

// APIResponse wraps the standard API response
type APIResponse struct {
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message,omitempty"`
	Error   string          `json:"error,omitempty"`
}
