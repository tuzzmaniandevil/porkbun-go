package porkbun

import "context"

// NameServers represents an array of name server hostnames.
type NameServers []string

// GetNameServersRequest represents the request structure for retrieving name servers.
type GetNameServersRequest struct {
	BaseRequest // Embeds the BaseRequest to include API credentials
}

// GetNameServersResponse represents the response structure for retrieving name servers.
type GetNameServersResponse struct {
	BaseResponse
	NS NameServers `json:"ns"` // An array of name server hostnames
}

// UpdateNameServersRequest represents the request structure for updating name servers.
type UpdateNameServersRequest struct {
	BaseRequest
	NS NameServers `json:"ns"` // An array of name servers to update the domain with
}

// UpdateNameServersResponse represents the response structure for updating name servers.
type UpdateNameServersResponse struct {
	BaseResponse
}

// GetNameServers retrieves the current name servers for the specified domain.
func (s *DomainsService) GetNameServers(ctx context.Context, domain string) (*GetNameServersResponse, error) {
	path := domainPath("getNs", domain)
	request := &GetNameServersRequest{}

	response := &GetNameServersResponse{}
	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}

// UpdateNameServers updates the name servers for the specified domain.
func (s *DomainsService) UpdateNameServers(ctx context.Context, domain string, newNameservers *NameServers) (*UpdateNameServersResponse, error) {
	path := domainPath("updateNs", domain)
	request := &UpdateNameServersRequest{
		NS: *newNameservers,
	}

	response := &UpdateNameServersResponse{}
	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}

// Interface guards to ensure the request types implement ApiKeyAcceptor.
var (
	_ ApiKeyAcceptor = (*GetNameServersRequest)(nil)
	_ ApiKeyAcceptor = (*UpdateNameServersRequest)(nil)
)
