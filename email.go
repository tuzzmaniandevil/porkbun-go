package porkbun

import "context"

// EmailService provides methods to interact with the email hosting API.
type EmailService struct {
	client *Client // Client used to communicate with the API
}

// setEmailPasswordRequest represents the request structure for setting an email password.
type setEmailPasswordRequest struct {
	baseRequest
	EmailAddress string `json:"emailAddress"` // The full email address, e.g. user@example.com
	Password     string `json:"password"`     // The new password, must pass Porkbun's validation rules
}

// SetEmailPasswordResponse represents the response structure for setting an email password.
type SetEmailPasswordResponse struct {
	BaseResponse
}

// SetPassword sets the password for an email hosting account on a domain
// managed by the API key.
func (s *EmailService) SetPassword(ctx context.Context, emailAddress string, password string, opts ...RequestOption) (*SetEmailPasswordResponse, error) {
	request := &setEmailPasswordRequest{
		EmailAddress: emailAddress,
		Password:     password,
	}

	response := &SetEmailPasswordResponse{}
	_, err := s.client.post(ctx, apiPath("email", "setPassword"), request, response, opts...)
	return response, err
}
