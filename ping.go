package porkbun

import "context"

// PingResponse represents the response structure for the Ping API.
type PingResponse struct {
	BaseResponse
	YourIP string `json:"yourIp"` // The IP address of the client making the request.
}

// PingRequest represents the request structure for the Ping API.
type PingRequest struct {
	BaseRequest
}

// Ping pings the Porkbun API to check its availability and returns the client's IP address.
func (s *Client) Ping(ctx context.Context) (*PingResponse, error) {
	request := &PingRequest{}
	response := &PingResponse{}

	// Make a POST request to the /ping endpoint
	resp, err := s.post(ctx, "/ping", request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}
