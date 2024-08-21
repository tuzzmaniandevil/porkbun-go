package porkbun

import (
	"context"
	"fmt"
)

// SslService provides methods to interact with the SSL certificate management API.
type SslService struct {
	client *Client // Client used to communicate with the API
}

// sslPath constructs the path for SSL-related API endpoints.
func sslPath(action string, a ...any) string {
	path := fmt.Sprintf("/ssl/%v", action)
	buildPathSegments(&path, a...)
	return path
}

// SslRetrieveRequest represents the request structure for retrieving an SSL certificate.
type SslRetrieveRequest struct {
	BaseRequest // Embeds the BaseRequest to include API credentials
}

// SslRetrieveResponse represents the response structure for an SSL certificate retrieval,
// including the certificate chain, private key, and public key.
type SslRetrieveResponse struct {
	BaseResponse

	Certificatechain string `json:"certificatechain"` // The complete certificate chain
	Privatekey       string `json:"privatekey"`       // The private key
	Publickey        string `json:"publickey"`        // The public key
}

// Retrieve fetches the SSL certificate bundle for the specified domain.
func (s *SslService) Retrieve(ctx context.Context, domain string) (*SslRetrieveResponse, error) {
	// Construct the API path
	path := sslPath("retrieve", domain)

	// Initialize the request and response structures
	request := &SslRetrieveRequest{}
	response := &SslRetrieveResponse{}

	// Make a POST request to the SSL retrieve endpoint
	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	// Attach the HTTP response to the SslRetrieveResponse
	response.HTTPResponse = resp
	return response, err
}

// Interface guards to ensure SslRetrieveRequest implements ApiKeyAcceptor
var (
	_ ApiKeyAcceptor = (*SslRetrieveRequest)(nil)
)
