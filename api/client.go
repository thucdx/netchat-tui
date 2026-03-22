package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrUnauthorized is returned when the server responds with HTTP 401.
var ErrUnauthorized = errors.New("api: unauthorized (401)")

// Client is an HTTP client for the Mattermost-compatible API.
type Client struct {
	baseURL string
	token   string
	userID  string
	http    *http.Client
}

// NewClient creates a new API Client. baseURL must use the https scheme (security
// requirement S3). The token is never included in returned error messages.
func NewClient(baseURL, token, userID string) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("api: invalid baseURL: %w", err)
	}
	if parsed.Scheme != "https" {
		return nil, fmt.Errorf("api: baseURL must use https scheme, got %q", parsed.Scheme)
	}

	// Strip trailing slash so path concatenation is predictable.
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		baseURL: baseURL,
		token:   token,
		userID:  userID,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// do executes an HTTP request against the API. The token is set as a Bearer
// Authorization header and is never included in any returned error message.
func (c *Client) do(method, path string, body io.Reader) ([]byte, error) {
	fullURL := c.baseURL + path

	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("api: building request for %s %s: %w", method, path, err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api: executing request for %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("api: reading response body for %s %s: %w", method, path, err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("%w", ErrUnauthorized)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api: %s %s returned status %d", method, path, resp.StatusCode)
	}

	return respBody, nil
}

// Get performs an HTTP GET request to the given path.
func (c *Client) Get(path string) ([]byte, error) {
	return c.do(http.MethodGet, path, nil)
}

// Post performs an HTTP POST request to the given path with the provided body.
func (c *Client) Post(path string, body io.Reader) ([]byte, error) {
	return c.do(http.MethodPost, path, body)
}
