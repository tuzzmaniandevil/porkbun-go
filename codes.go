package porkbun

// ErrorCode is the machine-readable code returned alongside an error. Branch on
// it rather than string-matching Message, which is meant for display.
//
// The API may add codes at any time, so treat an unrecognised value as a
// generic failure instead of assuming the set below is exhaustive.
type ErrorCode string

// Error lets an ErrorCode be used as an errors.Is target, which is how a caller
// matches one without unwrapping the response:
//
//	if errors.Is(err, porkbun.ErrCodeRateLimitExceeded) {
//	    // back off
//	}
//
// An ErrorCode is not itself a failure the SDK returns; only *ErrorResponse is.
func (c ErrorCode) Error() string { return string(c) }

// Authentication and protocol codes.
const (
	ErrCodeInvalidProtocol      ErrorCode = "INVALID_PROTOCOL"      // Request was not made over HTTPS
	ErrCodeMethodNotAllowed     ErrorCode = "METHOD_NOT_ALLOWED"    // HTTP method not allowed for this endpoint
	ErrCodeInvalidOrEmptyJSON   ErrorCode = "INVALID_OR_EMPTY_JSON" // Body is missing or not valid JSON
	ErrCodeAPIKeyRequired       ErrorCode = "API_KEY_REQUIRED"      // No API key or token was provided
	ErrCodeInvalidAPIKeys       ErrorCode = "INVALID_API_KEYS_001"  // Key and secret combination is invalid
	ErrCodeInvalidAPIKeysSecret ErrorCode = "INVALID_API_KEYS_002"  // Secret is wrong (deliberately vague)
	ErrCodeMissingSecretAPIKey  ErrorCode = "MISSING_SECRETAPIKEY"  // Secret missing or misnamed; the field is secretapikey
	ErrCodeInvalidToken         ErrorCode = "INVALID_TOKEN"         // Bearer token is invalid or expired
	ErrCodeInvalidUser          ErrorCode = "INVALID_USER"          // Account not found or not active
	ErrCodeIPNotAllowed         ErrorCode = "IP_NOT_ALLOWED"        // Source IP is not in the key's allowlist
	ErrCodeDomainNotAllowed     ErrorCode = "DOMAIN_NOT_ALLOWED"    // Target domain is not in the key's allowlist
)

// Rate limiting and idempotency codes.
const (
	ErrCodeRateLimitExceeded      ErrorCode = "RATE_LIMIT_EXCEEDED"      // See TTLRemaining and X-RateLimit-Reset
	ErrCodeIdempotencyKeyMismatch ErrorCode = "IDEMPOTENCY_KEY_MISMATCH" // Same key reused with a different body
	ErrCodeIdempotencyKeyInUse    ErrorCode = "IDEMPOTENCY_KEY_IN_USE"   // Original request is still in flight
)

// Domain operation codes.
const (
	ErrCodeInvalidDomain      ErrorCode = "INVALID_DOMAIN"       // Domain is invalid or not in your account
	ErrCodeDomainNotFound     ErrorCode = "DOMAIN_NOT_FOUND"     // Domain is not in the authenticated account
	ErrCodeDomainNotAvailable ErrorCode = "DOMAIN_NOT_AVAILABLE" // Domain is not available for registration
	ErrCodeInsufficientFunds  ErrorCode = "INSUFFICIENT_FUNDS"   // Account credit does not cover the purchase
)

// DNS operation codes.
const (
	ErrCodeInvalidType     ErrorCode = "INVALID_TYPE"      // DNS record type is not supported
	ErrCodeInvalidRecordID ErrorCode = "INVALID_RECORD_ID" // Record not found or not owned by your account
)

// API key authorization flow codes.
const (
	ErrCodeCodeVerifierRequired ErrorCode = "CODE_VERIFIER_REQUIRED" // PKCE request needs its codeVerifier
	ErrCodeInvalidCodeVerifier  ErrorCode = "INVALID_CODE_VERIFIER"  // codeVerifier does not match the challenge
	ErrCodeSecretAlreadyClaimed ErrorCode = "SECRET_ALREADY_CLAIMED" // The secret was already retrieved once
	ErrCodeRequestExpired       ErrorCode = "REQUEST_EXPIRED"        // Authorization request lapsed; start a new one
)

// NextActionType is a stable category telling a caller how to recover from an
// error. Branch on it instead of parsing the hint text.
type NextActionType string

// Recovery actions the API suggests.
const (
	NextActionFixRequest        NextActionType = "fix_request"        // Correct the request and resend
	NextActionAuthenticate      NextActionType = "authenticate"       // Fix or supply credentials
	NextActionEnableSetting     NextActionType = "enable_setting"     // Turn on a setting, e.g. per-domain API access
	NextActionVerifyAccount     NextActionType = "verify_account"     // Verify the account email or phone
	NextActionAddFunds          NextActionType = "add_funds"          // Top up the account balance
	NextActionRetry             NextActionType = "retry"              // Retry immediately
	NextActionWaitAndRetry      NextActionType = "wait_and_retry"     // Wait, then retry
	NextActionUseWebsite        NextActionType = "use_website"        // The operation is not available via the API
	NextActionChooseAlternative NextActionType = "choose_alternative" // Pick a different domain or value
	NextActionContactSupport    NextActionType = "contact_support"    // Escalate to Porkbun support
	NextActionNone              NextActionType = "none"               // No specific recovery applies
)

// WebhookEventType identifies a subscribable account event. Subscribe to exact
// types, to a prefix wildcard such as "dns.*", or to WebhookEventAll.
//
// Call WebhookService.EventTypes for the live catalog; new types can appear
// without a major version bump.
type WebhookEventType string

// Event types a webhook endpoint can subscribe to.
const (
	WebhookEventAll WebhookEventType = "*" // Every event type, including ones added later

	WebhookEventDomainRegistered        WebhookEventType = "domain.registered"
	WebhookEventDomainRenewed           WebhookEventType = "domain.renewed"
	WebhookEventDomainTransferCompleted WebhookEventType = "domain.transfer.completed"
	WebhookEventDomainExpiring          WebhookEventType = "domain.expiring" // Fires 60, 30 and 5 days out

	WebhookEventDNSRecordCreated WebhookEventType = "dns.record.created"
	WebhookEventDNSRecordUpdated WebhookEventType = "dns.record.updated"
	WebhookEventDNSRecordDeleted WebhookEventType = "dns.record.deleted"

	WebhookEventTest WebhookEventType = "webhook.test" // Sent by WebhookService.Test
)

// WebhookStatus is the delivery state of a webhook endpoint.
type WebhookStatus string

// Enum values for WebhookStatus.
const (
	WebhookStatusActive   WebhookStatus = "ACTIVE"   // Receiving deliveries
	WebhookStatusDisabled WebhookStatus = "DISABLED" // Paused, either manually or after repeated failures
)

// DeliveryStatus is the state of a single webhook delivery attempt.
type DeliveryStatus string

// Enum values for DeliveryStatus.
const (
	DeliveryPending    DeliveryStatus = "PENDING"    // Queued, not yet attempted
	DeliveryProcessing DeliveryStatus = "PROCESSING" // Attempt in flight
	DeliveryDelivered  DeliveryStatus = "DELIVERED"  // Acknowledged with a 2xx
	DeliveryFailed     DeliveryStatus = "FAILED"     // Gave up after MaxAttempts
)

// YesNo is a flag the API expresses as the strings "yes" and "no", which it does
// for the URL forward options and the listing filters. The zero value is unset:
// on a filter that means "do not filter", and on a request field it means "take
// the API's own default".
//
// It is distinct from AutoRenewStatus because the two vocabularies differ: the
// auto-renew filter on ListDomains takes yes/no, while UpdateAutoRenew takes
// on/off.
type YesNo string

// Enum values for YesNo.
const (
	Yes YesNo = "yes"
	No  YesNo = "no"
)

// CloudflareStatus reports whether the Cloudflare proxy is enabled for a domain.
type CloudflareStatus string

// Enum values for CloudflareStatus.
const (
	CloudflareEnabled  CloudflareStatus = "enabled"
	CloudflareDisabled CloudflareStatus = "disabled"
)

// InviteState is the state of an account registration invite.
type InviteState string

// Enum values for InviteState.
const (
	InvitePending  InviteState = "PENDING"  // The invite URL has not been used yet
	InviteAccepted InviteState = "ACCEPTED" // The user completed registration
	InviteExpired  InviteState = "EXPIRED"  // Not used within 48 hours, or canceled
)

// DeliveryMode is the flow an API key authorization request uses to hand over
// the secret key.
type DeliveryMode string

// Enum values for DeliveryMode.
const (
	DeliveryModeLegacy DeliveryMode = "legacy" // Secret shown in the browser, copied by hand
	DeliveryModePKCE   DeliveryMode = "pkce"   // Secret returned once to the holder of the code verifier
)

// DryRunOperation names the operation a dry run previewed.
type DryRunOperation string

// Enum values for DryRunOperation.
const (
	OperationRegistration DryRunOperation = "registration"
	OperationRenewal      DryRunOperation = "renewal"
	OperationTransfer     DryRunOperation = "transfer"
)

// SortDirection is the direction of a sorted listing.
type SortDirection string

// Enum values for SortDirection.
const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

// DomainSortField is a field ListDomains can sort by.
type DomainSortField string

// Enum values for DomainSortField.
const (
	DomainSortByDomain     DomainSortField = "domain"
	DomainSortByTLD        DomainSortField = "tld"
	DomainSortByCreateDate DomainSortField = "create_date"
	DomainSortByExpireDate DomainSortField = "expire_date"
)

// MarketplaceSortField is a field MarketplaceService.ListDomains can sort by.
type MarketplaceSortField string

// Enum values for MarketplaceSortField.
const (
	MarketplaceSortByDomain    MarketplaceSortField = "domain"
	MarketplaceSortByTLD       MarketplaceSortField = "tld"
	MarketplaceSortByPrice     MarketplaceSortField = "price"
	MarketplaceSortBySLDLength MarketplaceSortField = "sld_length"
)

// TermsAgreement confirms agreement to the Domain Name Registration Agreement
// and the product terms, which CreateDomain requires. The API accepts "yes" or
// "1"; use TermsAgreed.
type TermsAgreement string

// TermsAgreed is the value CreateDomain requires to register.
const TermsAgreed TermsAgreement = "yes"

// AutoRenewStatus is the auto-renew setting UpdateAutoRenew applies.
type AutoRenewStatus string

// Enum values for AutoRenewStatus.
const (
	AutoRenewOn  AutoRenewStatus = "on"
	AutoRenewOff AutoRenewStatus = "off"
)
