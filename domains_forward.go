package porkbun

import (
	"context"
	"errors"
)

// ForwardType is a custom type for specifying the type of URL forward.
type ForwardType string

// Constants representing valid values for ForwardType.
const (
	Temporary ForwardType = "temporary"
	Permanent ForwardType = "permanent"
	Masked    ForwardType = "masked"
)

// RedirectType is the exact redirect code stored for a URL forward.
// It distinguishes a 302 from a 307, which both report as Temporary.
type RedirectType string

// Constants representing valid values for RedirectType.
const (
	Redirect301    RedirectType = "301"
	Redirect302    RedirectType = "302"
	Redirect307    RedirectType = "307"
	RedirectMasked RedirectType = "masked"
)

// URLForward represents the details of a URL forward for a domain.
type URLForward struct {
	Subdomain    string       `json:"subdomain"`              // Optional subdomain to forward, empty if forwarding the root domain
	Location     string       `json:"location"`               // The destination URL for the forward
	Type         ForwardType  `json:"type"`                   // The type of forward: "temporary", "permanent" or "masked"
	RedirectType RedirectType `json:"redirectType,omitempty"` // Exact redirect code, takes precedence over Type when set
	IncludePath  YesNo        `json:"includePath"`            // Whether to include the URI path in the forward
	Wildcard     YesNo        `json:"wildcard"`               // Whether to forward all subdomains of the forwarded subdomain
}

// URLForwardRecord represents the data structure for a URL forward, including its ID.
type URLForwardRecord struct {
	ID string `json:"id"` // The ID of the URL forward
	URLForward
}

// getDomainURLForwardingRequest represents the request structure for retrieving domain URL forwards.
type getDomainURLForwardingRequest struct {
	baseRequest // Embeds the baseRequest to include API credentials
}

// ListURLForwardsResponse represents the response structure for retrieving domain URL forwards.
type ListURLForwardsResponse struct {
	BaseResponse
	Forwards []URLForwardRecord `json:"forwards"` // List of URL forwards for the domain
}

// addDomainURLForwardRequest represents the request structure for adding a new URL forward.
type addDomainURLForwardRequest struct {
	baseRequest
	*URLForward // Embeds the URLForward to include the forward details
}

// AddURLForwardResponse represents the response structure for adding a new URL forward.
type AddURLForwardResponse struct {
	BaseResponse
}

// deleteDomainURLForwardRequest represents the request structure for deleting a URL forward.
type deleteDomainURLForwardRequest struct {
	baseRequest
}

// DeleteURLForwardResponse represents the response structure for deleting a URL forward.
type DeleteURLForwardResponse struct {
	BaseResponse
}

// ListURLForwards retrieves the list of URL forwards for a specified domain.
func (s *DomainsService) ListURLForwards(ctx context.Context, domain string) (*ListURLForwardsResponse, error) {
	path := apiPath("domain", "getUrlForwarding", domain)

	request := &getDomainURLForwardingRequest{}
	response := &ListURLForwardsResponse{}
	_, err := s.client.post(ctx, path, request, response)
	return response, err
}

// AddURLForward adds a new URL forward for the specified domain.
//
// Type is required by the API. When only RedirectType is set it is derived:
// 301 is permanent, 302 and 307 are temporary, masked is masked.
//
// Subdomain is the subdomain alone, empty for the root domain. A fully qualified
// name loses the domain again, because the API accepts only alphanumerics and
// hyphens here and would reject the dotted form.
func (s *DomainsService) AddURLForward(ctx context.Context, domain string, forwardAttributes *URLForward, opts ...RequestOption) (*AddURLForwardResponse, error) {
	if forwardAttributes == nil {
		return &AddURLForwardResponse{}, errors.New("porkbun: AddURLForward requires the forward to add, at least Location and Type")
	}

	path := apiPath("domain", "addUrlForward", domain)

	forward := *forwardAttributes
	forward.Subdomain = unqualifyName(forward.Subdomain, domain)
	switch {
	case forward.Type != "":
	case forward.RedirectType == "":
		// type is required and has no meaningful zero value. temporary is the
		// API's own default for an unspecified forward.
		forward.Type = Temporary
	case forward.RedirectType == Redirect301:
		forward.Type = Permanent
	case forward.RedirectType == RedirectMasked:
		forward.Type = Masked
	default:
		forward.Type = Temporary // 302 and 307 are both temporary redirects
	}

	// Both are required as "yes" or "no", and have no meaningful zero value to
	// send, so an unset field takes the API's own default.
	if forward.IncludePath == "" {
		forward.IncludePath = No
	}
	if forward.Wildcard == "" {
		forward.Wildcard = No
	}

	request := &addDomainURLForwardRequest{
		URLForward: &forward,
	}

	response := &AddURLForwardResponse{}
	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}

// DeleteURLForward deletes a URL forward for the specified domain by record ID.
func (s *DomainsService) DeleteURLForward(ctx context.Context, domain string, recordID string, opts ...RequestOption) (*DeleteURLForwardResponse, error) {
	path := apiPath("domain", "deleteUrlForward", domain, recordID)

	request := &deleteDomainURLForwardRequest{}
	response := &DeleteURLForwardResponse{}
	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}
