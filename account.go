package porkbun

import (
	"context"
	"net/url"
)

// AccountService provides methods to interact with the account API.
type AccountService struct {
	client *Client // Client used to communicate with the API
}

// BalanceResponse represents the response structure for the account balance.
type BalanceResponse struct {
	BaseResponse
	Balance int64  `json:"balance"` // Available account credit balance in cents
	Display string `json:"display"` // Human-readable balance string, e.g. "$12.34"
}

// APISettings holds the account's API spend control settings. All amounts are in cents.
type APISettings struct {
	MonthlySpendLimit *int64 `json:"monthlySpendLimit"` // Maximum API spend per calendar month, nil means no limit
	LowBalanceAlert   *int64 `json:"lowBalanceAlert"`   // Alert when the balance drops below this amount, nil disables it
	AutoTopup         bool   `json:"autoTopup"`         // Whether automatic balance top-up is enabled
	TopupThreshold    *int64 `json:"topupThreshold"`    // Balance that triggers an auto top-up, nil disables it
	TopupAmount       *int64 `json:"topupAmount"`       // Amount added during an auto top-up
}

// APISettingsResponse represents the response structure for the API spend settings.
type APISettingsResponse struct {
	BaseResponse
	Settings     APISettings `json:"settings"`     // The account's API spend control settings
	MonthlySpend int64       `json:"monthlySpend"` // Total API spend in the current calendar month, in cents
}

// inviteRequest represents the request structure for creating an account invite.
type inviteRequest struct {
	baseRequest
	InviteOptions
}

// InviteResponse represents the response structure for creating an account invite.
type InviteResponse struct {
	BaseResponse
	InviteToken string `json:"inviteToken"` // Opaque token, pass to InviteStatus to track completion
	InviteURL   string `json:"inviteUrl"`   // URL to send to the prospective user
	Expires     string `json:"expires"`     // UTC datetime when the invite expires, 48 hours from creation
}

// InviteStatusResponse represents the response structure for checking an invite.
type InviteStatusResponse struct {
	BaseResponse
	InviteStatus InviteState `json:"inviteStatus"`           // InvitePending, InviteAccepted or InviteExpired
	NewAccountID FlexInt64   `json:"newAccountId,omitempty"` // ID of the created account, present when InviteAccepted
}

// GetBalance returns the available account credit balance.
func (s *AccountService) GetBalance(ctx context.Context) (*BalanceResponse, error) {
	response := &BalanceResponse{}

	_, err := s.client.get(ctx, "/account/balance", response)
	return response, err
}

// GetAPISettings returns the account's API spend control settings and the
// current calendar month's spend total.
func (s *AccountService) GetAPISettings(ctx context.Context) (*APISettingsResponse, error) {
	response := &APISettingsResponse{}

	_, err := s.client.get(ctx, "/account/apiSettings", response)
	return response, err
}

// InviteOptions provides the optional details for an account registration invite.
type InviteOptions struct {
	Email     string `json:"email,omitempty"`     // Email address to pre-fill on the registration form
	ReturnURL string `json:"returnUrl,omitempty"` // HTTPS URL to redirect the user to after registration
}

// CreateInvite generates a one-time account registration invite. The invite
// expires after 48 hours and each token can only be used once.
func (s *AccountService) CreateInvite(ctx context.Context, options *InviteOptions, opts ...RequestOption) (*InviteResponse, error) {
	request := &inviteRequest{}

	if options != nil {
		request.InviteOptions = *options
	}

	response := &InviteResponse{}
	_, err := s.client.post(ctx, "/account/invite", request, response, opts...)
	return response, err
}

// InviteStatus returns the current status of a registration invite created by
// the calling API key.
func (s *AccountService) InviteStatus(ctx context.Context, token string) (*InviteStatusResponse, error) {
	params := url.Values{}
	params.Set("token", token)

	response := &InviteStatusResponse{}
	_, err := s.client.get(ctx, buildQuery("/account/inviteStatus", params), response)
	return response, err
}
