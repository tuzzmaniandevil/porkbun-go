package porkbun

import (
	"context"
	"encoding/json"
	"fmt"
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
	Code          string `json:"code"`            // Coupon code
	MaxPerUser    int64  `json:"max_per_user"`    // Maximum number of uses per user
	FirstYearOnly string `json:"first_year_only"` // Indicates if the coupon is applicable only for the first year
	Type          string `json:"type"`            // Type of discount (e.g., amount, percentage)
	Amount        int64  `json:"amount"`          // Discount amount
}

// Coupons is a custom type that represents either a map of coupon codes to Coupon details
// or an empty array, which is unmarshaled as a nil map.
type Coupons map[string]Coupon

// UnmarshalJSON handles the unmarshaling of the Coupons field, which can be either
// a map of coupons or an empty array. This method ensures correct parsing based on the input type.
func (c *Coupons) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a map of coupons
	var couponsMap map[string]Coupon
	if err := json.Unmarshal(data, &couponsMap); err == nil {
		*c = couponsMap
		return nil
	}

	// Try to unmarshal as an empty array
	var emptyArray []interface{}
	if err := json.Unmarshal(data, &emptyArray); err == nil && len(emptyArray) == 0 {
		*c = nil // Set Coupons to nil if it's an empty array
		return nil
	}

	// Return an error if neither format is valid
	return fmt.Errorf("coupons field has an unexpected type")
}

// PricingResponse wraps the response from the pricing API, including the base response
// and the pricing details for various domain types.
type PricingResponse struct {
	BaseResponse
	Pricing map[string]Pricing `json:"pricing"` // Map of domain type to pricing details
}

// PricingService provides methods to interact with the pricing API.
type PricingService struct {
	client *Client // Client used to communicate with the API
}

// ListPricing retrieves the pricing information for various domain types from the API.
// It returns a PricingResponse containing the parsed pricing data.
func (s *PricingService) ListPricing(ctx context.Context) (*PricingResponse, error) {
	// Initialize an empty PricingResponse
	response := &PricingResponse{}

	// Make a POST request to the pricing endpoint
	resp, err := s.client.post(ctx, "/pricing/get", nil, response)
	if err != nil {
		return nil, err
	}

	// Attach the HTTP response to the PricingResponse
	response.HTTPResponse = resp
	return response, nil
}

// Interface guards ensure that the Coupons type implements the json.Unmarshaler interface.
var (
	_ json.Unmarshaler = (*Coupons)(nil)
)
