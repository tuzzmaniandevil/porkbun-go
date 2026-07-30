package porkbun

import (
	"context"
)

// SSLService provides methods to interact with the SSL certificate management API.
type SSLService struct {
	client *Client // Client used to communicate with the API
}

// sslRetrieveRequest represents the request structure for retrieving an SSL certificate.
type sslRetrieveRequest struct {
	baseRequest // Embeds the baseRequest to include API credentials
}

// SSLRetrieveResponse represents the response structure for an SSL certificate retrieval,
// including the certificate chain, private key, and public key.
type SSLRetrieveResponse struct {
	BaseResponse

	CertificateChain string `json:"certificatechain"` // The complete certificate chain
	PrivateKey       string `json:"privatekey"`       // The private key
	PublicKey        string `json:"publickey"`        // The public key
}

// Retrieve fetches the SSL certificate bundle for the specified domain.
func (s *SSLService) Retrieve(ctx context.Context, domain string) (*SSLRetrieveResponse, error) {
	path := apiPath("ssl", "retrieve", domain)

	request := &sslRetrieveRequest{}
	response := &SSLRetrieveResponse{}

	_, err := s.client.post(ctx, path, request, response)
	return response, err
}
