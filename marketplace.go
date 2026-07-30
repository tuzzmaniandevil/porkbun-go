package porkbun

import "context"

// MarketplaceService provides methods to interact with the marketplace API.
type MarketplaceService struct {
	client *Client // Client used to communicate with the API
}

// MarketplaceDomain represents a domain listed on the Porkbun marketplace.
type MarketplaceDomain struct {
	CreateDate string  `json:"create_date"` // Date the listing was created
	Domain     string  `json:"domain"`      // The listed domain name
	TLD        string  `json:"tld"`         // The top-level domain
	SLDLength  int64   `json:"sld_length"`  // Character length of the SLD
	Price      float64 `json:"price"`       // Listing price in USD
}

// listMarketplaceRequest represents the request structure for listing marketplace domains.
type listMarketplaceRequest struct {
	baseRequest
	MarketplaceListOptions
}

// ListMarketplaceResponse represents the response structure for listing marketplace domains.
type ListMarketplaceResponse struct {
	BaseResponse
	Count    int64               `json:"count"`    // Number of domains returned in this response
	Filtered bool                `json:"filtered"` // True when one or more filters were applied
	Domains  []MarketplaceDomain `json:"domains"`  // The marketplace listings
}

// MarketplaceListOptions provides options for filtering marketplace listings.
// Supplying any of Query, TLDs, SLDLengthMin, SLDLengthMax or SortName switches
// the API into filtered mode, which returns up to 1000 matches and ignores
// Start and Limit.
type MarketplaceListOptions struct {
	Start         *int64               `json:"start,omitempty"`         // Pagination offset, unfiltered mode only
	Limit         *int64               `json:"limit,omitempty"`         // Page size, unfiltered mode only. Default 1000, max 5000
	Query         *string              `json:"query,omitempty"`         // SLD substring search. Prefix a term with "-" to exclude it
	TLDs          []string             `json:"tlds,omitempty"`          // Limit to listings under these TLDs (without a leading dot)
	SLDLengthMin  *int64               `json:"sldLengthMin,omitempty"`  // Minimum SLD character length
	SLDLengthMax  *int64               `json:"sldLengthMax,omitempty"`  // Maximum SLD character length
	SortName      MarketplaceSortField `json:"sortName,omitempty"`      // Field to sort by
	SortDirection SortDirection        `json:"sortDirection,omitempty"` // SortAscending or SortDescending
}

// ListDomains retrieves domains listed on the Porkbun marketplace.
func (s *MarketplaceService) ListDomains(ctx context.Context, options *MarketplaceListOptions) (*ListMarketplaceResponse, error) {
	request := &listMarketplaceRequest{}

	if options != nil {
		request.MarketplaceListOptions = *options
	}

	response := &ListMarketplaceResponse{}
	_, err := s.client.post(ctx, "/marketplace/getAll", request, response)
	return response, err
}
