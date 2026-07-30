package porkbun

import "context"

// PingResponse represents the response structure for the Ping API.
type PingResponse struct {
	BaseResponse
	YourIP           string `json:"yourIp"`                     // The IP address of the client making the request.
	XForwardedFor    string `json:"xForwardedFor,omitempty"`    // Raw value of the X-Forwarded-For header.
	CredentialsValid bool   `json:"credentialsValid,omitempty"` // True when valid credentials were supplied.
}

// pingRequest represents the request structure for the Ping API.
type pingRequest struct {
	baseRequest
}

// Ping returns the caller's public IP address, and validates the client's
// credentials when it has any: CredentialsValid reports the outcome, and
// credentials the API rejects are returned as an error.
//
// Credentials are optional here, so a client configured without them sends
// none at all rather than empty ones. Use IP for an unconditionally
// credential-free lookup.
func (c *Client) Ping(ctx context.Context) (*PingResponse, error) {
	response := &PingResponse{}

	// With no credentials configured, send an empty body and no auth headers
	// rather than empty credentials the API would reject.
	if c.apiKey == "" && c.secret == "" {
		_, err := c.post(ctx, "/ping", &struct{}{}, response, withoutHeaderAuth())
		return response, err
	}

	_, err := c.post(ctx, "/ping", &pingRequest{}, response)
	return response, err
}

// IP returns the caller's public IP address. It never sends or validates
// credentials, which makes it the right call for a dynamic DNS client: an API
// key restricted to specific source IPs would otherwise be rejected from the
// very address you are trying to discover.
//
// Set Options.IPv4Only to resolve over IPv4 (api-ipv4.porkbun.com) when the
// host has both an IPv4 and an IPv6 route.
func (c *Client) IP(ctx context.Context) (*PingResponse, error) {
	response := &PingResponse{}

	_, err := c.get(ctx, "/ip", response, withoutHeaderAuth())
	return response, err
}
