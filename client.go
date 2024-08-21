package porkbun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// HTTPClient defines an interface for making HTTP requests.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Options defines the configuration options for the Porkbun API client.
type Options struct {
	HttpClient   *HTTPClient // Custom HTTP client, defaults to http.Client if nil.
	ApiKey       string      // Public API key provided by Porkbun.
	SecretApiKey string      // Secret API key provided by Porkbun.
	IPv4Only     bool        // If true, use IPv4-only base URL.
	UserAgent    string      // Custom User-Agent string, defaults to "porkbun-go/1.0.0".
}

// NewClient initializes a new Porkbun API client with the provided options.
func NewClient(options *Options) *Client {
	var httpClient HTTPClient

	if options.HttpClient != nil {
		httpClient = *options.HttpClient
	} else {
		httpClient = &http.Client{}
	}

	client := &Client{
		httpClient: &httpClient,
		apiKey:     options.ApiKey,
		secret:     options.SecretApiKey,
		userAgent:  options.UserAgent,
	}

	if options.IPv4Only {
		client.baseURL = ipv4OnlyBaseURL
	} else {
		client.baseURL = defaultBaseURL
	}

	client.Pricing = &PricingService{client: client}
	client.Domains = &DomainsService{client: client}
	client.Dns = &DnsService{client: client}
	client.Ssl = &SslService{client: client}

	return client
}

// Client represents a Porkbun API client.
type Client struct {
	httpClient *HTTPClient

	baseURL   string
	userAgent string

	secret string
	apiKey string

	// Services
	Pricing *PricingService
	Domains *DomainsService
	Dns     *DnsService
	Ssl     *SslService
}

// post is a helper method to make a POST request to the API.
func (c *Client) post(ctx context.Context, path string, payload interface{}, obj interface{}) (*http.Response, error) {
	return c.makeRequest(ctx, http.MethodPost, path, payload, obj)
}

// makeRequest creates and sends an HTTP request to the API, and handles the response.
func (c *Client) makeRequest(ctx context.Context, method, path string, payload interface{}, obj interface{}) (*http.Response, error) {
	req, err := c.newRequest(method, path, payload)
	if err != nil {
		return nil, err
	}

	resp, err := c.request(ctx, req, obj)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// newRequest creates a new HTTP request with the given method, path, and payload.
func (c *Client) newRequest(method, path string, payload interface{}) (*http.Request, error) {
	url := c.baseURL + path

	body := new(bytes.Buffer)
	if payload != nil {
		if pr, ok := payload.(ApiKeyAcceptor); ok {
			pr.SetCredentials(c.apiKey, c.secret)
		}

		if err := json.NewEncoder(body).Encode(payload); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", formatUserAgent(c.userAgent))

	return req, err
}

// request sends an HTTP request and decodes the response into the provided object.
func (c *Client) request(ctx context.Context, req *http.Request, obj interface{}) (*http.Response, error) {
	if ctx == nil {
		return nil, errors.New("context must be non-nil")
	}
	req = req.WithContext(ctx)

	resp, err := (*c.httpClient).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = checkResponse(resp)
	if err != nil {
		return resp, err
	}

	if obj == nil {
		return resp, nil
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(raw, obj)
	return resp, err
}

// checkResponse checks the HTTP response for errors.
func checkResponse(resp *http.Response) error {
	if code := resp.StatusCode; code == http.StatusOK {
		return nil
	}

	errorResponse := &ErrorResponse{}
	errorResponse.HTTPResponse = resp

	// Attempt to decode the response body into the errorResponse struct
	err := json.NewDecoder(resp.Body).Decode(errorResponse)
	if err != nil && err != io.EOF { // Allow for an empty body (EOF)
		// If decoding fails, return a generic error with the HTTP status code
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return errorResponse
}

// formatUserAgent formats the User-Agent string, appending the default if a custom one is provided.
func formatUserAgent(customUserAgent string) string {
	if customUserAgent == "" {
		return defaultUserAgent
	}
	return fmt.Sprintf("%s %s", customUserAgent, defaultUserAgent)
}
