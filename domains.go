package porkbun

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"time"
)

// DomainsService provides methods to interact with the domain management API.
type DomainsService struct {
	client *Client // Client used to communicate with the API
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
	APIAccess    BoolNumber `json:"apiAccess"`        // Indicates if the domain is opted in to API access (true/false)
	NotLocal     BoolNumber `json:"notLocal"`         // Indicates if the domain is not local (true/false)
	Labels       []Label    `json:"labels,omitempty"` // Optional labels associated with the domain
}

// domainTimeFormat is the format the API uses for the domain dates. It is not
// RFC 3339, so the dates need converting in both directions.
const domainTimeFormat = "2006-01-02 15:04:05"

// UnmarshalJSON handles the custom unmarshalling of the Domain struct, including parsing dates.
func (d *Domain) UnmarshalJSON(data []byte) error {
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

	// A date in some other shape costs that one field rather than the whole
	// response. Failing here would abort the enclosing array decode, so a single
	// malformed row in a 1000-domain ListDomains would lose the other 999 and
	// return a silently short list. An unparsed date reads as the zero time,
	// which a caller can test with IsZero.
	d.CreateDate, _ = time.Parse(domainTimeFormat, aux.CreateDate)
	d.ExpireDate, _ = time.Parse(domainTimeFormat, aux.ExpireDate)

	return nil
}

// MarshalJSON writes the dates back in the format the API sent them, rather than
// the RFC 3339 a time.Time encodes to by default, which the API does not accept.
func (d Domain) MarshalJSON() ([]byte, error) {
	type Alias Domain
	aux := struct {
		CreateDate string `json:"createDate"`
		ExpireDate string `json:"expireDate"`
		Alias
	}{
		Alias: Alias(d),
	}

	if !d.CreateDate.IsZero() {
		aux.CreateDate = d.CreateDate.Format(domainTimeFormat)
	}

	if !d.ExpireDate.IsZero() {
		aux.ExpireDate = d.ExpireDate.Format(domainTimeFormat)
	}

	return json.Marshal(aux)
}

// Label represents a label that can be associated with a domain, including its ID, title, and color.
type Label struct {
	ID    string `json:"id"`    // The ID of the label
	Title string `json:"title"` // The title of the label
	Color string `json:"color"` // The color associated with the label
}

// listDomainsRequest represents the request structure for listing domains.
type listDomainsRequest struct {
	baseRequest
	DomainListOptions
}

// ListDomainsResponse represents the response structure for listing domains.
type ListDomainsResponse struct {
	BaseResponse
	Count   int64    `json:"count,omitempty"` // Number of domains returned in this page
	Domains []Domain `json:"domains"`         // Array of domains and their details
}

// DomainListOptions provides options for filtering the list of domains.
// All fields are optional and may be combined freely.
type DomainListOptions struct {
	Start              *int64          `json:"start,omitempty"`              // Zero-based offset for pagination, up to 1000 domains per call
	IncludeLabels      YesNo           `json:"includeLabels,omitempty"`      // Return label metadata for each domain
	Domain             *string         `json:"domain,omitempty"`             // Exact domain name match, returns 0 or 1 result
	NameContains       *string         `json:"nameContains,omitempty"`       // Case-insensitive substring match against the full domain name
	ExpiringWithinDays *int64          `json:"expiringWithinDays,omitempty"` // Only domains expiring within this many days
	TLDs               []string        `json:"tlds,omitempty"`               // Limit to these TLDs (without a leading dot)
	AutoRenew          YesNo           `json:"autoRenew,omitempty"`          // Filter by auto-renew state. Yes or No, not AutoRenewOn/Off
	APIAccess          YesNo           `json:"apiAccess,omitempty"`          // Filter by API access opt-in
	SortName           DomainSortField `json:"sortName,omitempty"`           // Field to sort by
	SortDirection      SortDirection   `json:"sortDirection,omitempty"`      // SortAscending or SortDescending
}

// ListDomains retrieves a list of domains associated with the account, with optional filters for pagination and labels.
func (s *DomainsService) ListDomains(ctx context.Context, options *DomainListOptions) (*ListDomainsResponse, error) {
	path := apiPath("domain", "listAll")
	request := &listDomainsRequest{}

	if options != nil {
		request.DomainListOptions = *options
	}

	response := &ListDomainsResponse{}
	_, err := s.client.post(ctx, path, request, response)
	return response, err
}

// GetDomainResponse represents the response structure for retrieving a single domain.
type GetDomainResponse struct {
	BaseResponse
	Domain Domain `json:"domain"` // The domain and its details
}

// GetDomainOptions provides the optional parameters for GetDomain.
type GetDomainOptions struct {
	IncludeLabels YesNo // Return label metadata. The zero value takes the API's default of no.
}

// GetDomain retrieves the metadata for a single domain in the account. Pass nil
// for the API's defaults.
//
//	resp, err := client.Domains.GetDomain(ctx, "example.com",
//	    &porkbun.GetDomainOptions{IncludeLabels: porkbun.Yes})
func (s *DomainsService) GetDomain(ctx context.Context, domain string, options *GetDomainOptions) (*GetDomainResponse, error) {
	params := url.Values{}
	if options != nil && options.IncludeLabels != "" {
		params.Set("includeLabels", string(options.IncludeLabels))
	}
	path := buildQuery(apiPath("domain", "get", domain), params)

	response := &GetDomainResponse{}
	_, err := s.client.get(ctx, path, response)
	return response, err
}

// updateAutoRenewRequest represents the request structure for updating the auto-renew setting.
type updateAutoRenewRequest struct {
	baseRequest
	Status  AutoRenewStatus `json:"status"`            // Auto-renew status to set
	Domains []string        `json:"domains,omitempty"` // Additional domains to update, combined with the one in the path
}

// UpdateAutoRenewResult represents the per-domain outcome of an auto-renew update.
type UpdateAutoRenewResult struct {
	Status  string `json:"status"`            // Per-domain status
	Message string `json:"message,omitempty"` // Per-domain message
}

// UpdateAutoRenewResults maps a domain name to the outcome of its auto-renew
// update. An update that matched no domain decodes as a nil map.
type UpdateAutoRenewResults map[string]UpdateAutoRenewResult

// UnmarshalJSON implements custom unmarshalling logic for UpdateAutoRenewResults.
func (r *UpdateAutoRenewResults) UnmarshalJSON(data []byte) error {
	return unmarshalFlexMap(data, r, "auto-renew results")
}

// UpdateAutoRenewResponse represents the response structure for updating the auto-renew setting.
type UpdateAutoRenewResponse struct {
	BaseResponse
	Results UpdateAutoRenewResults `json:"results"` // Outcome keyed by domain name
}

// UpdateAutoRenew turns auto-renew on or off for one or more domains, and
// reports the outcome per domain in Results.
//
//	client.Domains.UpdateAutoRenew(ctx, porkbun.AutoRenewOn, []string{"example.com"})
func (s *DomainsService) UpdateAutoRenew(ctx context.Context, status AutoRenewStatus, domains []string, opts ...RequestOption) (*UpdateAutoRenewResponse, error) {
	response := &UpdateAutoRenewResponse{}

	if len(domains) == 0 {
		return response, errors.New("porkbun: UpdateAutoRenew requires at least one domain")
	}

	// The API takes one domain in the path and any others in the body, and
	// combines them. The trailing slash on the bulk-only form is deliberate: the
	// API serves an HTML 404 for the path without it.
	path := apiPath("domain", "updateAutoRenew", domains[0])

	request := &updateAutoRenewRequest{
		Status:  status,
		Domains: domains[1:],
	}

	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}

var (
	_ json.Unmarshaler = (*Domain)(nil)
	_ json.Marshaler   = Domain{}
	_ json.Unmarshaler = (*UpdateAutoRenewResults)(nil)
)
