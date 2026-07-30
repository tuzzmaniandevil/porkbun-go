package porkbun

import (
	"context"
	"errors"
)

// NameServers represents an array of name server hostnames.
type NameServers []string

// getNameServersRequest represents the request structure for retrieving name servers.
type getNameServersRequest struct {
	baseRequest // Embeds the baseRequest to include API credentials
}

// GetNameServersResponse represents the response structure for retrieving name servers.
type GetNameServersResponse struct {
	BaseResponse
	NS NameServers `json:"ns"` // An array of name server hostnames
}

// updateNameServersRequest represents the request structure for updating name servers.
type updateNameServersRequest struct {
	baseRequest
	dryRunnable
	NS NameServers `json:"ns"` // An array of name servers to update the domain with
}

// UpdateNameServersResponse represents the response structure for updating name servers.
// When the request ran with WithDryRun, DryRun is true and nothing was changed.
type UpdateNameServersResponse struct {
	BaseResponse
	DryRunResult
}

// GetNameServers retrieves the current name servers for the specified domain.
func (s *DomainsService) GetNameServers(ctx context.Context, domain string) (*GetNameServersResponse, error) {
	path := apiPath("domain", "getNs", domain)
	request := &getNameServersRequest{}

	response := &GetNameServersResponse{}
	_, err := s.client.post(ctx, path, request, response)
	return response, err
}

// UpdateNameServers replaces the name servers for the specified domain.
func (s *DomainsService) UpdateNameServers(ctx context.Context, domain string, newNameservers NameServers, opts ...RequestOption) (*UpdateNameServersResponse, error) {
	if len(newNameservers) == 0 {
		return &UpdateNameServersResponse{}, errors.New("porkbun: UpdateNameServers requires the name servers to set")
	}

	path := apiPath("domain", "updateNs", domain)
	request := &updateNameServersRequest{
		NS: newNameservers,
	}

	response := &UpdateNameServersResponse{}
	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}
