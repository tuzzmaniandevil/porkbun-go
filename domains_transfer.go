package porkbun

import "context"

// TransferStatus represents the state of an inbound domain transfer.
type TransferStatus string

// Enum values for TransferStatus.
const (
	TransferNew             TransferStatus = "NEW"
	TransferPendingAuth     TransferStatus = "PENDINGAUTH"
	TransferPendingSubmit   TransferStatus = "PENDINGSUBMIT"
	TransferPendingTransfer TransferStatus = "PENDINGTRANSFER"
	TransferDone            TransferStatus = "DONE"
	TransferCanceled        TransferStatus = "CANCELED"
	TransferInit            TransferStatus = "INIT"
)

// Transfer represents an inbound domain transfer record.
type Transfer struct {
	Domain            string         `json:"domain"`                      // The domain being transferred
	Status            TransferStatus `json:"status"`                      // Current transfer state
	StatusDescription string         `json:"statusDescription,omitempty"` // Human-readable status
	TransferDate      string         `json:"transferDate,omitempty"`      // When the transfer was initiated
	OrderID           FlexInt64      `json:"orderId,omitempty"`           // Internal Porkbun order ID
}

// transferDomainRequest represents the request structure for initiating a transfer.
type transferDomainRequest struct {
	baseRequest
	dryRunnable
	AuthCode string `json:"authCode"` // The EPP auth code for the domain
	Cost     int64  `json:"cost"`     // Transfer cost in pennies, must match the pricing API exactly
}

// TransferDomainResponse represents the response structure for initiating a transfer.
// When the request ran with WithDryRun, the embedded DryRunPreview fields are
// populated instead of OrderID and nothing was charged.
type TransferDomainResponse struct {
	BaseResponse
	DryRunPreview
	Domain       string      `json:"domain"`                 // The domain being transferred
	Cost         int64       `json:"cost,omitempty"`         // The total amount charged in pennies, or the amount a dry run would charge
	OrderID      FlexInt64   `json:"orderId,omitempty"`      // Internal Porkbun order ID
	TransferID   FlexInt64   `json:"transferId,omitempty"`   // Internal transfer ID
	Balance      int64       `json:"balance,omitempty"`      // Remaining account credit balance in pennies
	TTLRemaining int64       `json:"ttlRemaining,omitempty"` // Seconds until the success rate limit window resets
	Limits       *RateLimits `json:"limits,omitempty"`       // Current rate limit state
}

// GetTransferResponse represents the response structure for a single transfer lookup.
type GetTransferResponse struct {
	BaseResponse
	Transfer Transfer `json:"transfer"` // The most recent transfer record for the domain
}

// ListTransfersResponse represents the response structure for listing active transfers.
type ListTransfersResponse struct {
	BaseResponse
	Transfers []Transfer `json:"transfers"` // Active inbound transfers, excluding completed and canceled ones
}

// TransferDomain initiates an inbound domain transfer to Porkbun using account
// credit. Transfers are processed asynchronously and typically take 5-7 days.
// Use WithDryRun to validate without initiating the transfer.
func (s *DomainsService) TransferDomain(ctx context.Context, domain string, authCode string, cost int64, opts ...RequestOption) (*TransferDomainResponse, error) {
	path := apiPath("domain", "transfer", domain)

	request := &transferDomainRequest{
		AuthCode: authCode,
		Cost:     cost,
	}
	response := &TransferDomainResponse{}

	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}

// GetTransfer returns the most recent transfer record for a domain in the account.
func (s *DomainsService) GetTransfer(ctx context.Context, domain string) (*GetTransferResponse, error) {
	path := apiPath("domain", "getTransfer", domain)

	response := &GetTransferResponse{}
	_, err := s.client.get(ctx, path, response)
	return response, err
}

// ListTransfers returns all active inbound transfers for the account.
func (s *DomainsService) ListTransfers(ctx context.Context) (*ListTransfersResponse, error) {
	path := apiPath("domain", "listTransfers")

	response := &ListTransfersResponse{}
	_, err := s.client.get(ctx, path, response)
	return response, err
}
