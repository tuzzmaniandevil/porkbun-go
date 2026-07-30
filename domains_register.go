package porkbun

import (
	"context"
	"errors"
)

// RateLimit describes the state of a single rate limit window.
type RateLimit struct {
	TTL             int64  `json:"TTL"`             // Window in seconds
	Limit           int64  `json:"limit"`           // Maximum operations allowed in the window
	Used            int64  `json:"used"`            // Operations used so far in the window
	NaturalLanguage string `json:"naturalLanguage"` // Human-readable summary
}

// RateLimits describes the attempt and success rate limit windows returned by
// the billable domain operations.
type RateLimits struct {
	Attempts RateLimit `json:"attempts"` // Attempt rate limit state
	Success  RateLimit `json:"success"`  // Successful-operation rate limit state
}

// DryRunPreview holds the fields returned when a write is run with WithDryRun.
// Nothing was created and no charge was made.
//
// The booleans carry no omitempty: false is a meaningful answer here. A caller
// runs a dry run precisely to learn that WouldSucceed is false, so it has to
// survive being marshalled back out.
type DryRunPreview struct {
	DryRun          bool            `json:"dryRun"`                // Always true on a dry-run preview
	WouldSucceed    bool            `json:"wouldSucceed"`          // Whether the operation would complete
	Operation       DryRunOperation `json:"operation,omitempty"`   // OperationRegistration, OperationRenewal or OperationTransfer
	TLD             string          `json:"tld,omitempty"`         // The top-level domain
	Available       string          `json:"available,omitempty"`   // Availability as reported by the registry check
	Premium         bool            `json:"premium"`               // Whether the domain is a premium/aftermarket name
	Duration        int64           `json:"duration,omitempty"`    // Term in years that would be purchased
	CostDisplay     string          `json:"costDisplay,omitempty"` // Human-readable cost
	SufficientFunds bool            `json:"sufficientFunds"`       // Whether the balance covers the cost
	// The three spend-cap fields are pointers because the API sends them only
	// when a cap is configured, and a plain false would be indistinguishable from
	// "this operation would breach the cap". WouldSucceed is the go/no-go.
	MonthlySpendLimit       *int64 `json:"monthlySpendLimit,omitempty"`       // Monthly API spend cap in pennies, nil when no cap is configured
	MonthlySpendSoFar       *int64 `json:"monthlySpendSoFar,omitempty"`       // API spend so far this month in pennies, nil when no cap is configured
	WithinMonthlySpendLimit *bool  `json:"withinMonthlySpendLimit,omitempty"` // Whether the cost stays within the cap, nil when no cap is configured
}

// checkDomainRequest represents the request structure for a domain availability check.
type checkDomainRequest struct {
	baseRequest
}

// DomainPrice represents a price quote for one product type.
type DomainPrice struct {
	Type         string `json:"type"`                   // The price type, e.g. registration, renewal or transfer
	Price        string `json:"price"`                  // Price per year in USD
	RegularPrice string `json:"regularPrice,omitempty"` // Standard (non-promo) price in USD
}

// DomainAvailability holds the availability and pricing details for a domain.
type DomainAvailability struct {
	Avail          YesNo         `json:"avail"`                    // Whether the domain is available for registration
	Type           string        `json:"type"`                     // The price type, always "registration" for the primary price
	Price          string        `json:"price"`                    // Registration price per year in USD
	FirstYearPromo YesNo         `json:"firstYearPromo,omitempty"` // Whether the price is a first-year promotion
	RegularPrice   string        `json:"regularPrice,omitempty"`   // Standard (non-promo) registration price in USD
	Premium        YesNo         `json:"premium,omitempty"`        // Whether this is a premium domain
	MinDuration    int64         `json:"minDuration,omitempty"`    // Minimum registration duration in years required by the registry
	Additional     DomainPricing `json:"additional"`               // Renewal and transfer pricing for this domain
}

// DomainPricing holds the renewal and transfer quotes returned alongside a
// registration price.
type DomainPricing struct {
	Renewal  DomainPrice `json:"renewal"`  // Renewal pricing for this domain
	Transfer DomainPrice `json:"transfer"` // Transfer pricing for this domain
}

// CheckDomainResponse represents the response structure for a domain availability check.
type CheckDomainResponse struct {
	BaseResponse
	Response     DomainAvailability `json:"response"`               // Availability and pricing details
	Limits       *RateLimit         `json:"limits,omitempty"`       // Current rate limit usage for this account
	TTLRemaining int64              `json:"ttlRemaining,omitempty"` // Seconds until the rate limit window resets
}

// CheckDomain checks whether a domain is available for registration and returns
// current registration, renewal and transfer pricing.
func (s *DomainsService) CheckDomain(ctx context.Context, domain string) (*CheckDomainResponse, error) {
	path := apiPath("domain", "checkDomain", domain)

	request := &checkDomainRequest{}
	response := &CheckDomainResponse{}

	_, err := s.client.post(ctx, path, request, response)
	return response, err
}

// createDomainRequest represents the request structure for registering a domain.
type createDomainRequest struct {
	baseRequest
	dryRunnable
	CreateDomainOptions
}

// CreateDomainResponse represents the response structure for registering a domain.
// When the request ran with WithDryRun, the embedded DryRunPreview fields are
// populated instead of OrderID and nothing was charged.
type CreateDomainResponse struct {
	BaseResponse
	DryRunPreview
	Domain       string      `json:"domain"`                 // The registered domain name
	Cost         int64       `json:"cost,omitempty"`         // The total amount charged in pennies
	OrderID      FlexInt64   `json:"orderId,omitempty"`      // Internal Porkbun order ID
	Balance      int64       `json:"balance,omitempty"`      // Remaining account credit balance in pennies
	TTLRemaining int64       `json:"ttlRemaining,omitempty"` // Seconds until the success rate limit window resets
	Limits       *RateLimits `json:"limits,omitempty"`       // Current rate limit state
}

// CreateDomainOptions provides the details required to register a domain.
type CreateDomainOptions struct {
	Cost         int64          `json:"cost"`                   // Registration cost in pennies, must equal the current price exactly
	AgreeToTerms TermsAgreement `json:"agreeToTerms"`           // Must be TermsAgreed to confirm agreement to the registration terms
	WhoisPrivacy *bool          `json:"whoisPrivacy,omitempty"` // Override the account-level WHOIS privacy default for this registration
}

// CreateDomain registers a domain using account credit. Registrations are always
// for the registry-minimum duration, and cost must match the price returned by
// CheckDomain in pennies. Use WithDryRun to validate without registering.
func (s *DomainsService) CreateDomain(ctx context.Context, domain string, options *CreateDomainOptions, opts ...RequestOption) (*CreateDomainResponse, error) {
	if options == nil {
		return &CreateDomainResponse{}, errors.New("porkbun: CreateDomain requires options, at least Cost and AgreeToTerms")
	}

	path := apiPath("domain", "create", domain)

	request := &createDomainRequest{CreateDomainOptions: *options}
	response := &CreateDomainResponse{}

	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}

// renewDomainRequest represents the request structure for renewing a domain.
type renewDomainRequest struct {
	baseRequest
	dryRunnable
	Cost int64 `json:"cost"` // Renewal cost in pennies, must equal the current renewal price exactly
}

// RenewDomainResponse represents the response structure for renewing a domain.
// When the request ran with WithDryRun, the embedded DryRunPreview fields are
// populated instead of OrderID and nothing was charged.
type RenewDomainResponse struct {
	BaseResponse
	DryRunPreview
	Domain         string      `json:"domain"`                   // The renewed domain name
	ExpirationDate string      `json:"expirationDate,omitempty"` // The new expiration date returned by the registry
	Cost           int64       `json:"cost,omitempty"`           // The total amount charged in pennies
	OrderID        FlexInt64   `json:"orderId,omitempty"`        // Internal Porkbun order ID
	Balance        int64       `json:"balance,omitempty"`        // Remaining account credit balance in pennies
	TTLRemaining   int64       `json:"ttlRemaining,omitempty"`   // Seconds until the success rate limit window resets
	Limits         *RateLimits `json:"limits,omitempty"`         // Current rate limit state
}

// RenewDomain renews a domain using account credit for the registry-minimum
// duration. cost must match the current renewal price in pennies, which
// CheckDomain returns under Response.Additional.Renewal.
// Use WithDryRun to validate without renewing.
func (s *DomainsService) RenewDomain(ctx context.Context, domain string, cost int64, opts ...RequestOption) (*RenewDomainResponse, error) {
	path := apiPath("domain", "renew", domain)

	request := &renewDomainRequest{Cost: cost}
	response := &RenewDomainResponse{}

	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}

// RegistrationRequirementsResponse represents the machine-readable registration
// requirements for a TLD.
type RegistrationRequirementsResponse struct {
	BaseResponse
	TLD                       string  `json:"tld"`                                 // The TLD these requirements apply to
	APIRegisterable           bool    `json:"apiRegisterable"`                     // Whether the TLD can be registered via the API
	RegistrationDurationYears int64   `json:"registrationDurationYears,omitempty"` // Fixed registration term the API uses
	MaxRegistrationYears      *int64  `json:"maxRegistrationYears,omitempty"`      // Maximum years the registry allows, nil when the registry does not specify one
	WhoisPrivacySupported     bool    `json:"whoisPrivacySupported"`               // Whether WHOIS privacy is supported
	RequiresValidatedAddress  bool    `json:"requiresValidatedAddress"`            // Whether a validated address is required
	RegistrantOnly            bool    `json:"registrantOnly"`                      // TLD uses only the registrant contact
	RequestSchema             RawJSON `json:"requestSchema,omitempty"`             // JSON Schema for the /domain/create request body
	RegistryRequirements      RawJSON `json:"registryRequirements,omitempty"`      // JSON Schema of extra registry eligibility fields, absent when the TLD has none
	NotAPIRegisterableReason  string  `json:"notApiRegisterableReason,omitempty"`  // Why the TLD is not API-registerable
}

// GetRegistrationRequirements returns the registration requirements for a TLD
// (without a leading dot), including the /domain/create body as a JSON Schema.
// Call it before CreateDomain to know upfront whether and how a TLD can be registered.
func (s *DomainsService) GetRegistrationRequirements(ctx context.Context, tld string) (*RegistrationRequirementsResponse, error) {
	path := apiPath("domain", "getRegistrationRequirements", tld)

	response := &RegistrationRequirementsResponse{}
	_, err := s.client.get(ctx, path, response)
	return response, err
}
