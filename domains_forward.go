package porkbun

import "context"

// ForwardType is a custom type for specifying the type of URL forward.
type ForwardType string

// Constants representing valid values for ForwardType.
const (
	Temporary ForwardType = "temporary"
	Permanent ForwardType = "permanent"
)

// UrlForward represents the details of a URL forward for a domain.
type UrlForward struct {
	Subdomain   string      `json:"subdomain"`   // Optional subdomain to forward, empty if forwarding the root domain
	Location    string      `json:"location"`    // The destination URL for the forward
	Type        ForwardType `json:"type"`        // The type of forward: "temporary" or "permanent"
	IncludePath string      `json:"includePath"` // Whether to include the URI path in the forward: "yes" or "no"
	Wildcard    string      `json:"wildcard"`    // Whether to forward all subdomains: "yes" or "no"`
}

// UrlForwardData represents the data structure for a URL forward, including its ID.
type UrlForwardData struct {
	Id string `json:"id"` // The ID of the URL forward
	UrlForward
}

// GetDomainURLForwardingRequest represents the request structure for retrieving domain URL forwards.
type GetDomainURLForwardingRequest struct {
	BaseRequest // Embeds the BaseRequest to include API credentials
}

// GetDomainURLForwardingResponse represents the response structure for retrieving domain URL forwards.
type GetDomainURLForwardingResponse struct {
	BaseResponse
	Forwards []UrlForwardData `json:"forwards"` // List of URL forwards for the domain
}

// AddDomainUrlForwardRequest represents the request structure for adding a new URL forward.
type AddDomainUrlForwardRequest struct {
	BaseRequest
	*UrlForward // Embeds the UrlForward to include the forward details
}

// AddDomainUrlForwardResponse represents the response structure for adding a new URL forward.
type AddDomainUrlForwardResponse struct {
	BaseResponse
}

// DeleteDomainUrlForwardRequest represents the request structure for deleting a URL forward.
type DeleteDomainUrlForwardRequest struct {
	BaseRequest
}

// DeleteDomainUrlForwardResponse represents the response structure for deleting a URL forward.
type DeleteDomainUrlForwardResponse struct {
	BaseResponse
}

// GetDomainURLForwarding retrieves the list of URL forwards for a specified domain.
func (s *DomainsService) GetDomainURLForwarding(ctx context.Context, domain string) (*GetDomainURLForwardingResponse, error) {
	path := domainPath("getUrlForwarding", domain)

	request := &GetDomainURLForwardingRequest{}
	response := &GetDomainURLForwardingResponse{}
	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}

// AddDomainUrlForward adds a new URL forward for the specified domain.
func (s *DomainsService) AddDomainUrlForward(ctx context.Context, domain string, forwardAttributes *UrlForward) (*AddDomainUrlForwardResponse, error) {
	path := domainPath("addUrlForward", domain)

	request := &AddDomainUrlForwardRequest{
		UrlForward: forwardAttributes,
	}

	response := &AddDomainUrlForwardResponse{}
	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}

// DeleteDomainUrlForward deletes a URL forward for the specified domain by record ID.
func (s *DomainsService) DeleteDomainUrlForward(ctx context.Context, domain string, recordId string) (*DeleteDomainUrlForwardResponse, error) {
	path := domainPath("deleteUrlForward", domain, recordId)

	request := &DeleteDomainUrlForwardRequest{}
	response := &DeleteDomainUrlForwardResponse{}
	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}

// Interface guards to ensure that the required interfaces are implemented.
var (
	_ ApiKeyAcceptor = (*GetDomainURLForwardingRequest)(nil)
	_ ApiKeyAcceptor = (*AddDomainUrlForwardRequest)(nil)
	_ ApiKeyAcceptor = (*DeleteDomainUrlForwardRequest)(nil)
)
