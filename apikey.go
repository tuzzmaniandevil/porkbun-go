package porkbun

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// APIKeyService provides methods for the API key authorization flow, which mints
// credentials for an account that has none configured yet.
type APIKeyService struct {
	client *Client // Client used to communicate with the API
}

// apiKeyRequestRequest represents the request structure for initiating an API key
// authorization flow. No credentials are required.
type apiKeyRequestRequest struct {
	Name                string `json:"name,omitempty"`                // Human-readable name shown to the account holder on the approval screen
	CodeChallenge       string `json:"codeChallenge,omitempty"`       // PKCE challenge, base64url(SHA-256(codeVerifier))
	CodeChallengeMethod string `json:"codeChallengeMethod,omitempty"` // PKCE challenge method, only "S256" is supported
}

// APIKeyRequestResponse represents the response structure for initiating an API
// key authorization flow.
type APIKeyRequestResponse struct {
	BaseResponse
	RequestToken string       `json:"requestToken"`           // Token used to poll ApiKeyRetrieve for approval
	AuthURL      string       `json:"authUrl"`                // URL the account holder must visit to approve the request
	Expiration   string       `json:"expiration"`             // ISO datetime when this request expires, 10 minutes from creation
	DeliveryMode DeliveryMode `json:"deliveryMode,omitempty"` // DeliveryModePKCE when a code challenge was supplied, otherwise DeliveryModeLegacy
}

// apiKeyRetrieveRequest represents the request structure for polling an API key
// authorization request.
type apiKeyRetrieveRequest struct {
	RequestToken string `json:"requestToken"`           // The token returned by ApiKeyRequest
	CodeVerifier string `json:"codeVerifier,omitempty"` // PKCE verifier, required when the request used a code challenge
}

// APIKeyRetrieveResponse represents the response structure for polling an API key
// authorization request. Status is PENDING while awaiting approval.
type APIKeyRetrieveResponse struct {
	BaseResponse
	APIKey       string `json:"apikey,omitempty"`       // The approved public API key, present when status is SUCCESS
	SecretAPIKey string `json:"secretapikey,omitempty"` // The secret API key, returned once for a PKCE request only
}

// PKCECodeChallenge returns the base64url-encoded SHA-256 challenge for a verifier,
// as required by APIKeyRequestOptions.CodeVerifier.
func PKCECodeChallenge(codeVerifier string) string {
	sum := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// NewPKCECodeVerifier generates a random 43-character PKCE code verifier.
// Keep it secret: whoever presents it to ApiKeyRetrieve receives the secret API key.
func NewPKCECodeVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// APIKeyRequestOptions provides the optional details for an API key authorization request.
type APIKeyRequestOptions struct {
	Name string // Human-readable name shown to the account holder on the approval screen

	// CodeVerifier opts into the PKCE flow. Its challenge is sent with the
	// request, and passing the same verifier to ApiKeyRetrieve returns both the
	// public and secret keys once, so nothing has to be copied from a browser.
	// Use NewPKCECodeVerifier to generate one.
	CodeVerifier string
}

// Request initiates an API key authorization flow and returns a request
// token plus an approval URL for the account holder to visit. The URL is valid
// for 10 minutes. Poll ApiKeyRetrieve afterwards to collect the key.
//
// No credentials are required, so this works on a client with no keys configured.
func (s *APIKeyService) Request(ctx context.Context, options *APIKeyRequestOptions) (*APIKeyRequestResponse, error) {
	request := &apiKeyRequestRequest{}

	if options != nil {
		request.Name = options.Name
		if options.CodeVerifier != "" {
			request.CodeChallenge = PKCECodeChallenge(options.CodeVerifier)
			request.CodeChallengeMethod = "S256"
		}
	}

	response := &APIKeyRequestResponse{}
	_, err := s.client.post(ctx, "/apikey/request", request, response, withoutHeaderAuth())
	return response, err
}

// Retrieve polls whether the account holder has approved an authorization
// request. Status is PENDING until they do. Pass the codeVerifier used to create
// a PKCE request to also receive the secret key, which is only ever returned once.
func (s *APIKeyService) Retrieve(ctx context.Context, requestToken string, codeVerifier string) (*APIKeyRetrieveResponse, error) {
	request := &apiKeyRetrieveRequest{
		RequestToken: requestToken,
		CodeVerifier: codeVerifier,
	}

	response := &APIKeyRetrieveResponse{}
	_, err := s.client.post(ctx, "/apikey/retrieve", request, response, withoutHeaderAuth())
	return response, err
}
