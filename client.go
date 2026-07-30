package porkbun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HTTPClient defines an interface for making HTTP requests.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Options defines the configuration options for the Porkbun API client.
type Options struct {
	HTTPClient   HTTPClient // Custom HTTP client, defaults to an http.Client with DefaultTimeout if nil.
	APIKey       string     // Public API key provided by Porkbun.
	SecretAPIKey string     // Secret API key provided by Porkbun.
	IPv4Only     bool       // If true, use IPv4-only base URL.
	UserAgent    string     // Custom User-Agent string, prepended to the default "porkbun-go/<Version>".

	// BaseURL overrides the API endpoint, for a test server, a proxy, or a
	// recording harness. Empty selects the Porkbun endpoint, IPv4-only or not
	// according to IPv4Only. A trailing slash is trimmed.
	BaseURL string
}

// NewClient initializes a new Porkbun API client with the provided options.
// Pass nil, or the zero Options, for a client with no credentials: enough for
// Ping, IP and ListPricing.
func NewClient(options *Options) *Client {
	if options == nil {
		options = &Options{}
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		// The zero http.Client waits forever, which turns a dropped connection
		// into a hung caller. Supply an HTTPClient to choose your own.
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}

	client := &Client{
		httpClient: httpClient,
		apiKey:     options.APIKey,
		secret:     options.SecretAPIKey,
		userAgent:  options.UserAgent,
	}

	switch {
	case options.BaseURL != "":
		client.baseURL = strings.TrimRight(options.BaseURL, "/")
	case options.IPv4Only:
		client.baseURL = ipv4OnlyBaseURL
	default:
		client.baseURL = defaultBaseURL
	}

	client.Pricing = &PricingService{client: client}
	client.Domains = &DomainsService{client: client}
	client.DNS = &DNSService{client: client}
	client.SSL = &SSLService{client: client}
	client.Account = &AccountService{client: client}
	client.Marketplace = &MarketplaceService{client: client}
	client.Email = &EmailService{client: client}
	client.Webhooks = &WebhookService{client: client}
	client.APIKey = &APIKeyService{client: client}

	return client
}

// Client represents a Porkbun API client.
type Client struct {
	httpClient HTTPClient

	baseURL   string
	userAgent string

	secret string
	apiKey string

	// Services
	Pricing     *PricingService
	Domains     *DomainsService
	DNS         *DNSService
	SSL         *SSLService
	Account     *AccountService
	Marketplace *MarketplaceService
	Email       *EmailService
	Webhooks    *WebhookService
	APIKey      *APIKeyService
}

// post is a helper method to make a POST request to the API.
func (c *Client) post(ctx context.Context, path string, payload any, obj any, opts ...RequestOption) (*http.Response, error) {
	return c.makeRequest(ctx, http.MethodPost, path, payload, obj, opts...)
}

// get is a helper method to make a GET request to the API.
// GET requests carry no body, so they authenticate with the API key headers.
func (c *Client) get(ctx context.Context, path string, obj any, opts ...RequestOption) (*http.Response, error) {
	return c.makeRequest(ctx, http.MethodGet, path, nil, obj, opts...)
}

// makeRequest creates and sends an HTTP request to the API, and handles the response.
func (c *Client) makeRequest(ctx context.Context, method, path string, payload any, obj any, opts ...RequestOption) (*http.Response, error) {
	req, err := c.newRequest(method, path, payload, newRequestConfig(opts))
	if err != nil {
		return nil, err
	}

	resp, err := c.request(ctx, req, obj)

	// Attach the HTTP response even when the call failed, so RateLimit,
	// APIVersion and IdempotentReplayed work on the returned response too and
	// not only on the error.
	if setter, ok := obj.(httpResponseSetter); ok && resp != nil {
		setter.setHTTPResponse(resp)
	}

	return resp, err
}

// httpResponseSetter is implemented by every response type through its embedded
// BaseResponse.
type httpResponseSetter interface {
	setHTTPResponse(resp *http.Response)
}

// newRequest creates a new HTTP request with the given method, path, and payload.
func (c *Client) newRequest(method, path string, payload any, cfg *requestConfig) (*http.Request, error) {
	url := c.baseURL + path

	if pathHasDotSegment(path) {
		return nil, fmt.Errorf("porkbun: %s %s: a path segment is \".\" or \"..\", which would retarget the request at another endpoint", method, path)
	}

	// The API ignores a dryRun it does not implement and performs the write, so a
	// rehearsal it cannot give is an error rather than the real thing.
	if cfg.dryRun {
		dr, ok := payload.(dryRunAcceptor)
		if !ok {
			return nil, fmt.Errorf("porkbun: %s %s does not support WithDryRun, and would perform the operation for real", method, path)
		}
		dr.SetDryRun(true)
	}

	body := new(bytes.Buffer)
	bodyHasCredentials := false
	if payload != nil {
		if pr, ok := payload.(apiKeyAcceptor); ok {
			pr.SetCredentials(c.apiKey, c.secret)
			bodyHasCredentials = true
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

	// The API applies header auth only when the body carries no credentials, so
	// sending both would put the secret in a second place a proxy or a request log
	// could capture it, for no effect.
	//
	// Header auth also needs both halves: a key with an empty secret earns a
	// MISSING_SECRETAPIKEY rather than the anonymous access it would otherwise get.
	if c.apiKey != "" && c.secret != "" && !cfg.noHeaderAuth && !bodyHasCredentials {
		req.Header.Set("X-API-Key", c.apiKey)
		req.Header.Set("X-Secret-API-Key", c.secret)
	}

	if cfg.idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", cfg.idempotencyKey)
	}

	return req, nil
}

// request sends an HTTP request and decodes the response into the provided object.
func (c *Client) request(ctx context.Context, req *http.Request, obj any) (*http.Response, error) {
	if ctx == nil {
		return nil, errors.New("porkbun: nil context")
	}
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// A real *http.Client never returns either of these, but Options.HTTPClient
	// exists to be substituted, and a hand-written double that does should get an
	// error rather than a panic from inside the SDK.
	if resp == nil {
		return nil, errors.New("porkbun: HTTPClient returned a nil response and a nil error")
	}
	if resp.Body == nil {
		resp.Body = http.NoBody
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, fmt.Errorf("porkbun: %s %s: reading response body: %w", req.Method, req.URL, err)
	}

	// BaseResponse.HTTPResponse hands the caller this response, whose Body is now
	// drained and about to be closed. Leave a reader that reports EOF rather than
	// "read on closed response body".
	resp.Body = http.NoBody

	// Decoded before the status is judged, and kept even when it does not fully
	// decode, so a caller reading Status, Code and Message off the response sees
	// them whichever way a failure arrived: as a 4xx, or as ERROR inside a 200.
	var decodeErr error
	if obj != nil {
		decodeErr = json.Unmarshal(raw, obj)
	}

	if resp.StatusCode != http.StatusOK {
		return resp, errorFromStatus(raw, resp)
	}

	// A number of endpoints report a failure as status ERROR inside a 200,
	// including the DNS and glue writes. Returning that as success would have a
	// caller believe a record was deleted when it was not.
	//
	// An error body carries an error shape, whose fields can clash with the
	// success shape obj expects, so the reported error takes precedence over the
	// decode failure that clash produces: otherwise the code a caller branches on
	// is lost.
	if apiErr := apiErrorFrom(raw, resp); apiErr != nil {
		return resp, apiErr
	}

	if decodeErr != nil {
		// A gateway can answer 200 with an HTML interstitial, which is the likeliest
		// way a body fails to decode. Name the call, so the failure is attributable,
		// and keep %w so the *json.SyntaxError underneath stays matchable.
		return resp, fmt.Errorf("porkbun: %s %s: %d: decoding response body: %w",
			req.Method, req.URL, resp.StatusCode, decodeErr)
	}

	return resp, nil
}

// apiErrorFrom returns the error a 200 response body reports through its status
// field, and nil when the body reports no error or is not shaped like one.
func apiErrorFrom(raw []byte, resp *http.Response) *ErrorResponse {
	errorResponse := &ErrorResponse{}
	if err := json.Unmarshal(raw, errorResponse); err != nil {
		return nil
	}

	if errorResponse.Status != StatusError {
		return nil
	}

	errorResponse.HTTPResponse = resp
	if errorResponse.Message == "" {
		errorResponse.Message = "the API reported an error without a message"
	}

	return errorResponse
}

// errorFromStatus builds the error for a response whose status code is not 200,
// from the body bytes already read.
func errorFromStatus(raw []byte, resp *http.Response) error {
	errorResponse := &ErrorResponse{}
	errorResponse.HTTPResponse = resp

	// A gateway or WAF in front of the API can answer with HTML or plain text.
	// Keep returning an *ErrorResponse either way, so errors.As and the status
	// code stay available, and describe the status when there is no message.
	if err := json.Unmarshal(raw, errorResponse); err != nil && len(raw) > 0 {
		// Half-decoded fields cannot be trusted, so discard them.
		errorResponse.Message = ""
		errorResponse.Code = ""
		errorResponse.NextAction = nil
	}

	if errorResponse.Status == "" {
		errorResponse.Status = StatusError
	}

	// Error already prints the status code, so the fallback message only needs
	// to name it.
	if errorResponse.Message == "" {
		errorResponse.Message = http.StatusText(resp.StatusCode)
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
