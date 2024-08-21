package porkbun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// DomainsService provides methods to interact with the domain management API.
type DomainsService struct {
	client *Client // Client used to communicate with the API
}

// domainPath constructs the path for domain-related API endpoints.
func domainPath(action string, a ...any) string {
	path := fmt.Sprintf("/domain/%v", action)
	buildPathSegments(&path, a...)
	return path
}

// Domain represents the details of a domain, including its status, creation/expiration dates, and various settings.
type Domain struct {
	Domain       string     `json:"domain"`           // The domain name
	Status       string     `json:"status"`           // The status of the domain (e.g., ACTIVE)
	TLD          string     `json:"tld"`              // The top-level domain (TLD)
	CreateDate   time.Time  `json:"createDate"`       // The date the domain was created
	ExpireDate   time.Time  `json:"expireDate"`       // The date the domain will expire
	SecurityLock BoolString `json:"securityLock"`     // Indicates if the domain has a security lock (true/false)
	WhoisPrivacy BoolString `json:"whoisPrivacy"`     // Indicates if WHOIS privacy is enabled (true/false)
	AutoRenew    BoolNumber `json:"autoRenew"`        // Indicates if auto-renewal is enabled (true/false)
	NotLocal     BoolNumber `json:"notLocal"`         // Indicates if the domain is not local (true/false)
	Labels       []Label    `json:"labels,omitempty"` // Optional labels associated with the domain
}

// UnmarshalJSON handles the custom unmarshalling of the Domain struct, including parsing dates.
func (d *Domain) UnmarshalJSON(data []byte) error {
	// Custom time format used in the JSON
	const timeFormat = "2006-01-02 15:04:05"

	type Alias Domain
	aux := &struct {
		CreateDate string `json:"createDate"`
		ExpireDate string `json:"expireDate"`
		*Alias
	}{
		Alias: (*Alias)(d),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	d.CreateDate, err = time.Parse(timeFormat, aux.CreateDate)
	if err != nil {
		return fmt.Errorf("error parsing CreateDate: %w", err)
	}

	d.ExpireDate, err = time.Parse(timeFormat, aux.ExpireDate)
	if err != nil {
		return fmt.Errorf("error parsing ExpireDate: %w", err)
	}

	return nil
}

// Label represents a label that can be associated with a domain, including its ID, title, and color.
type Label struct {
	ID    string `json:"id"`    // The ID of the label
	Title string `json:"title"` // The title of the label
	Color string `json:"color"` // The color associated with the label
}

// ListDomainsRequest represents the request structure for listing domains.
type ListDomainsRequest struct {
	BaseRequest
	Start         *string `json:"start,omitempty"`         // Optional start index for pagination
	IncludeLabels *string `json:"includeLabels,omitempty"` // Optional flag to include label information
}

// ListDomainsResponse represents the response structure for listing domains.
type ListDomainsResponse struct {
	BaseResponse
	Domains []Domain `json:"domains"` // Array of domains and their details
}

// DomainListOptions provides options for filtering the list of domains.
type DomainListOptions struct {
	Start         *string // Optional start index for pagination
	IncludeLabels *string // Optional flag to include label information
}

// ListDomains retrieves a list of domains associated with the account, with optional filters for pagination and labels.
func (s *DomainsService) ListDomains(ctx context.Context, options *DomainListOptions) (*ListDomainsResponse, error) {
	path := domainPath("listAll")
	request := &ListDomainsRequest{}

	if options != nil {
		request.Start = options.Start
		request.IncludeLabels = options.IncludeLabels
	}

	response := &ListDomainsResponse{}
	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}

// Interface guards to ensure that the required interfaces are implemented.
var (
	_ json.Unmarshaler = (*Domain)(nil)
	_ ApiKeyAcceptor   = (*ListDomainsRequest)(nil)
)
