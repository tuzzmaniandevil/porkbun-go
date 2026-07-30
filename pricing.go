package porkbun

import (
	"context"
	"encoding/json"
)

// Pricing represents the pricing information for a domain,
// including registration, renewal, transfer costs, and any applicable coupons.
type Pricing struct {
	Registration string  `json:"registration"`          // Cost of domain registration
	Renewal      string  `json:"renewal"`               // Cost of domain renewal
	Transfer     string  `json:"transfer"`              // Cost of domain transfer
	Coupons      Coupons `json:"coupons,omitempty"`     // Applicable coupons, if any
	SpecialType  *string `json:"specialType,omitempty"` // Optional special pricing type
}

// Coupon represents the details of a coupon, such as the code, limits,
// applicability, and discount amount.
type Coupon struct {
	Code          string  `json:"code"`            // Coupon code
	MaxPerUser    int64   `json:"max_per_user"`    // Maximum number of uses per user
	FirstYearOnly YesNo   `json:"first_year_only"` // Whether the coupon applies to the first year only
	Type          string  `json:"type"`            // Type of discount (e.g., amount, percentage)
	Amount        float64 `json:"amount"`          // Discount amount
}

// Coupons maps a product type, such as "registration", to the coupon active for
// it. An absent coupon set decodes as a nil map.
type Coupons map[string]Coupon

// UnmarshalJSON implements custom unmarshalling logic for Coupons.
func (c *Coupons) UnmarshalJSON(data []byte) error {
	return unmarshalFlexMap(data, c, "coupons")
}

// PricingMap maps a TLD, without a leading dot, to its pricing.
type PricingMap map[string]Pricing

// UnmarshalJSON implements custom unmarshalling logic for PricingMap.
func (p *PricingMap) UnmarshalJSON(data []byte) error {
	return unmarshalFlexMap(data, p, "pricing")
}

// PricingResponse wraps the response from the pricing API, including the base response
// and the pricing details for various domain types.
type PricingResponse struct {
	BaseResponse
	Pricing PricingMap `json:"pricing"` // Map of domain type to pricing details
}

// pricingRequest represents the request structure for the pricing API.
// Pricing is public, so no credentials are sent.
type pricingRequest struct {
	TLDs []string `json:"tlds,omitempty"` // Optional TLDs to filter results by. All supported TLDs are returned when empty.
}

// PricingService provides methods to interact with the pricing API.
type PricingService struct {
	client *Client // Client used to communicate with the API
}

// ListPricing retrieves the pricing information for various domain types from the API.
// Pass one or more TLDs (without a leading dot) to filter the result.
func (s *PricingService) ListPricing(ctx context.Context, tlds ...string) (*PricingResponse, error) {
	response := &PricingResponse{}

	// Pricing is public, so no credentials are sent. TLDs is omitempty, so an
	// unfiltered call posts an empty object rather than an empty body.
	request := &pricingRequest{TLDs: tlds}

	_, err := s.client.post(ctx, "/pricing/get", request, response, withoutHeaderAuth())
	return response, err
}

var (
	_ json.Unmarshaler = (*Coupons)(nil)
	_ json.Unmarshaler = (*PricingMap)(nil)
)
