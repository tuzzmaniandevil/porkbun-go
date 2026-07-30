package porkbun

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Version is the SDK version, reported in the default User-Agent. Keep it in step
// with the release tag.
const Version = "1.1.0"

const (
	defaultBaseURL   = "https://api.porkbun.com/api/json/v3"
	ipv4OnlyBaseURL  = "https://api-ipv4.porkbun.com/api/json/v3"
	defaultUserAgent = "porkbun-go/" + Version
)

// DefaultTimeout is the request timeout on the HTTP client NewClient builds when
// Options.HTTPClient is nil. Pass your own HTTPClient to change it. The per-call
// context still applies, and a shorter deadline there wins.
const DefaultTimeout = 30 * time.Second

// apiKeyAcceptor is implemented by the request types that carry credentials in
// the body, which is how newRequest knows to fill them in.
type apiKeyAcceptor interface {
	SetCredentials(apiKey string, secretAPIKey string)
}

// dryRunAcceptor is implemented by the request types whose endpoint implements
// dryRun, which is how newRequest refuses WithDryRun on the others.
type dryRunAcceptor interface {
	SetDryRun(dryRun bool)
}

// baseRequest carries the credentials every authenticated request body needs.
type baseRequest struct {
	SecretAPIKey string `json:"secretapikey"` // The secret API key provided by Porkbun.
	APIKey       string `json:"apikey"`       // The public API key provided by Porkbun.
}

// SetCredentials sets the API and secret keys for the request.
func (br *baseRequest) SetCredentials(apiKey string, secretAPIKey string) {
	br.APIKey = apiKey
	br.SecretAPIKey = secretAPIKey
}

// dryRunnable is embedded by the request types whose endpoint implements dryRun,
// and only by those. WithDryRun fails on a request that does not embed it.
type dryRunnable struct {
	DryRun bool `json:"dryRun,omitempty"` // Validate only, do not perform the operation. See WithDryRun.
}

// SetDryRun marks the request as validate-only.
func (d *dryRunnable) SetDryRun(dryRun bool) {
	d.DryRun = dryRun
}

// RequestOption customizes a single write request. Options apply only to the
// call they are passed to, never to later calls.
type RequestOption func(*requestConfig)

// requestConfig holds the per-request settings built from the RequestOptions.
type requestConfig struct {
	dryRun         bool
	idempotencyKey string
	noHeaderAuth   bool
}

// WithDryRun runs every pre-flight check and reports what would happen, without
// creating, charging, or mutating anything.
//
// The API implements it on CreateDomain, RenewDomain, TransferDomain,
// UpdateNameServers and the DNS record writes. Passing it to any other call
// returns an error rather than performing the write, because the API would
// discard the flag and carry the operation out for real.
func WithDryRun() RequestOption {
	return func(cfg *requestConfig) {
		cfg.dryRun = true
	}
}

// DryRunResult holds the fields a non-billable write returns when run with
// WithDryRun: the DNS record writes and UpdateNameServers. Nothing was changed,
// and no record ID was allocated. The billable operations return the richer
// DryRunPreview instead.
type DryRunResult struct {
	DryRun       bool `json:"dryRun"`       // Always true on a dry-run preview
	WouldSucceed bool `json:"wouldSucceed"` // Whether the write would succeed
}

// WithIdempotencyKey sends the given key as the Idempotency-Key header. A retry
// with the same key within 24 hours replays the original response instead of
// repeating the operation, so a network failure cannot double-charge.
func WithIdempotencyKey(key string) RequestOption {
	return func(cfg *requestConfig) {
		cfg.idempotencyKey = key
	}
}

// withoutHeaderAuth suppresses the API key headers, for the /apikey flow that
// mints new credentials and accepts none.
func withoutHeaderAuth() RequestOption {
	return func(cfg *requestConfig) {
		cfg.noHeaderAuth = true
	}
}

// newRequestConfig applies the given options in order.
func newRequestConfig(opts []RequestOption) *requestConfig {
	cfg := &requestConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// BaseResponse represents the base structure for all API responses.
type BaseResponse struct {
	HTTPResponse *http.Response `json:"-"`                   // The underlying HTTP response, for its status code and headers.
	Status       string         `json:"status"`              // Status indicating whether the command was successfully processed.
	Message      string         `json:"message,omitempty"`   // Human-readable message. Present on ERROR, sometimes on SUCCESS.
	Code         ErrorCode      `json:"code,omitempty"`      // Machine-readable error code. Present when status is ERROR.
	RequestID    string         `json:"requestId,omitempty"` // Per-request UUID, taken from the body or the X-Request-Id header.
}

// Values BaseResponse.Status can carry. Only StatusError is treated as a
// failure; a call that returns StatusPending succeeded, and reports that the
// operation it started has not finished.
//
// These are untyped constants so they compare directly with Status:
//
//	if resp.Status == porkbun.StatusPending {
//	    // the user has not authorized the request yet
//	}
const (
	StatusSuccess = "SUCCESS"
	StatusError   = "ERROR"
	StatusPending = "PENDING" // Returned by ApiKeyRetrieve while authorization is outstanding
)

// setHTTPResponse records the HTTP response behind this API response, and fills
// in RequestID from the header when the body did not carry one. The API sets
// X-Request-Id on every response, including the ones whose body is not JSON, so
// this is what makes RequestID usable on exactly the failures worth reporting.
func (r *BaseResponse) setHTTPResponse(resp *http.Response) {
	r.HTTPResponse = resp

	if r.RequestID == "" && resp != nil {
		r.RequestID = resp.Header.Get("X-Request-Id")
	}
}

// RateLimitState reports the rate limit headers sent alongside a response.
// Only the API key authorization endpoints send these; elsewhere the limits
// appear in the response body, such as CheckDomainResponse.Limits.
type RateLimitState struct {
	Limit     int64 // X-RateLimit-Limit: requests allowed in the window
	Remaining int64 // X-RateLimit-Remaining: requests left in the window
	Reset     int64 // X-RateLimit-Reset: when the current window resets
}

// RateLimit returns the rate limit headers, and false when the response carried
// none.
//
//	if limits, ok := resp.RateLimit(); ok && limits.Remaining == 0 {
//	    // back off until limits.Reset
//	}
func (r *BaseResponse) RateLimit() (RateLimitState, bool) {
	if r.HTTPResponse == nil {
		return RateLimitState{}, false
	}

	header := r.HTTPResponse.Header

	// Any one of the three is enough: a 429 carries X-RateLimit-Reset for the
	// retry time and need not repeat the limit. A header the response did not
	// send reads 0, so test Reset before reading Remaining == 0 as exhaustion.
	if header.Get("X-RateLimit-Limit") == "" &&
		header.Get("X-RateLimit-Remaining") == "" &&
		header.Get("X-RateLimit-Reset") == "" {
		return RateLimitState{}, false
	}
	parse := func(name string) int64 {
		value, _ := strconv.ParseInt(header.Get(name), 10, 64)
		return value
	}

	return RateLimitState{
		Limit:     parse("X-RateLimit-Limit"),
		Remaining: parse("X-RateLimit-Remaining"),
		Reset:     parse("X-RateLimit-Reset"),
	}, true
}

// APIVersion returns the API specification version that served the request,
// taken from the X-API-Version header. It is empty if the header was absent.
func (r *BaseResponse) APIVersion() string {
	if r.HTTPResponse == nil {
		return ""
	}
	return r.HTTPResponse.Header.Get("X-API-Version")
}

// IdempotentReplayed reports whether the API replayed a stored response instead
// of performing the operation again, because the Idempotency-Key had been seen
// before.
func (r *BaseResponse) IdempotentReplayed() bool {
	if r.HTTPResponse == nil {
		return false
	}
	return r.HTTPResponse.Header.Get("Idempotent-Replayed") == "true"
}

// NextAction is a machine-readable remediation hint returned alongside many errors.
type NextAction struct {
	Type NextActionType `json:"type"`          // Stable action category, e.g. NextActionAddFunds.
	Hint string         `json:"hint"`          // Human-readable instruction.
	URL  string         `json:"url,omitempty"` // Where to take the action, when applicable.
}

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	BaseResponse
	NextAction   *NextAction `json:"next_action,omitempty"`  // How to recover from the error, when the API knows.
	TTLRemaining int64       `json:"ttlRemaining,omitempty"` // Seconds until the rate limit window resets, on RATE_LIMIT_EXCEEDED.
}

// Error implements the error interface for ErrorResponse.
func (r *ErrorResponse) Error() string {
	if r.HTTPResponse == nil {
		return "porkbun: " + r.Message
	}

	if r.HTTPResponse.Request == nil {
		return fmt.Sprintf("porkbun: %d: %s", r.HTTPResponse.StatusCode, r.Message)
	}

	// The query is dropped: InviteStatus carries the invite token there, and an
	// error string ends up in logs and bug reports.
	redacted := *r.HTTPResponse.Request.URL
	redacted.RawQuery = ""

	return fmt.Sprintf("porkbun: %s %s: %d: %s",
		r.HTTPResponse.Request.Method, redacted.String(),
		r.HTTPResponse.StatusCode, r.Message)
}

// Is reports whether this error carries the given ErrorCode, so a code can be
// matched as a sentinel with errors.Is as well as through errors.As:
//
//	if errors.Is(err, porkbun.ErrCodeInsufficientFunds) {
//	    // top up and retry
//	}
func (r *ErrorResponse) Is(target error) bool {
	code, ok := target.(ErrorCode)
	return ok && code != "" && code == r.Code
}

// String is a helper function that allocates a new string value and returns a pointer to it.
//
// Deprecated: use Ptr, which does the same for any type.
func String(v string) *string { return &v }

// Ptr returns a pointer to v, for the optional fields that take one. A literal
// has no address, so this is the way to pass one:
//
//	opts := &porkbun.DomainListOptions{
//	    NameContains:       porkbun.Ptr("example"),
//	    ExpiringWithinDays: porkbun.Ptr(int64(30)),
//	}
func Ptr[T any](v T) *T { return &v }

// unmarshalFlexMap decodes a JSON object into a map, and accepts the empty array
// the API sends in place of an empty object.
//
// Every field documented as an object arrives as [] when it has no entries: a
// domain with no DNSSEC records, an auto-renew update that matched nothing, a TLD
// with no coupons. The backend cannot tell an empty map from an empty list.
//
// what names the field, for the error message.
func unmarshalFlexMap[M ~map[string]V, V any](data []byte, m *M, what string) error {
	var array []json.RawMessage
	if err := json.Unmarshal(data, &array); err == nil {
		if len(array) != 0 {
			return fmt.Errorf("%s: expected an object or an empty array, got %d array elements", what, len(array))
		}
		*m = nil
		return nil
	}

	// The underlying map type, not M: decoding into M would re-enter this through
	// M's own UnmarshalJSON.
	var decoded map[string]V
	if err := json.Unmarshal(data, &decoded); err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}

	*m = decoded
	return nil
}

// unmarshalBool converts JSON data into a boolean value, in any of the three
// forms the API uses for one:
//
//	true and false
//	"1" for true, "0" or "" for false
//	1 for true, 0 for false
func unmarshalBool(data []byte) (bool, error) {
	var boolean bool
	if err := json.Unmarshal(data, &boolean); err == nil {
		return boolean, nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		if str == "" {
			return false, nil
		}
		parsedBool, err := strconv.ParseBool(str)
		if err != nil {
			return false, fmt.Errorf("parsing boolean: %s", str)
		}
		return parsedBool, nil
	}

	var num int
	if err := json.Unmarshal(data, &num); err == nil {
		switch num {
		case 0:
			return false, nil
		case 1:
			return true, nil
		default:
			return false, fmt.Errorf("parsing boolean: %d", num)
		}
	}

	return false, fmt.Errorf("invalid boolean format: %s", string(data))
}

// Bool is a boolean the API may send as a "1"/"0" string or a 1/0 number.
// Which form a given field uses is not consistent between endpoints, so this
// type accepts either.
type Bool bool

// UnmarshalJSON implements custom unmarshalling logic for Bool.
func (b *Bool) UnmarshalJSON(data []byte) error {
	parsedBool, err := unmarshalBool(data)
	if err != nil {
		return err
	}
	*b = Bool(parsedBool)
	return nil
}

// BoolString is an alias for Bool, which accepts both the string and the number form.
//
// Deprecated: use Bool.
type BoolString = Bool

// BoolNumber is an alias for Bool, which accepts both the string and the number form.
//
// Deprecated: use Bool.
type BoolNumber = Bool

// FlexInt64 is an integer the API may send as either a JSON number or a quoted
// string. Identifiers such as order and webhook ids arrive in both forms, so
// response fields use this type to accept either. A JSON null decodes as 0.
//
// Use Int64 to get a plain int64:
//
//	orderID := resp.OrderID.Int64()
type FlexInt64 int64

// UnmarshalJSON implements custom unmarshalling logic for FlexInt64.
func (i *FlexInt64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*i = 0
		return nil
	}

	var num int64
	if err := json.Unmarshal(data, &num); err == nil {
		*i = FlexInt64(num)
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return fmt.Errorf("invalid integer format: %s", string(data))
	}

	if str == "" {
		*i = 0
		return nil
	}

	parsed, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing integer: %s", str)
	}

	*i = FlexInt64(parsed)
	return nil
}

// Int64 returns the value as a plain int64.
func (i FlexInt64) Int64() int64 { return int64(i) }

// RawJSON is an undecoded JSON value from a field the API documents as nullable,
// such as the JSON Schemas GetRegistrationRequirements returns.
//
// Unlike json.RawMessage, which stores an explicit null as the four bytes "null",
// a null decodes to nil here, so an absent value is absent by either test:
//
//	if len(resp.RegistryRequirements) > 0 {
//	    // the TLD really does have registry requirements to read
//	}
type RawJSON json.RawMessage

// UnmarshalJSON implements custom unmarshalling logic for RawJSON.
func (r *RawJSON) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*r = nil
		return nil
	}

	*r = append((*r)[:0], data...)
	return nil
}

// MarshalJSON writes the value back verbatim, and an absent one as null.
func (r RawJSON) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return []byte("null"), nil
	}
	return r, nil
}

// Interface guards ensure the custom types implement json.Unmarshaler.
var (
	_ json.Unmarshaler = (*Bool)(nil)
	_ json.Unmarshaler = (*FlexInt64)(nil)
	_ json.Unmarshaler = (*RawJSON)(nil)
	_ json.Marshaler   = RawJSON(nil)
)

// buildQuery appends the given query parameters to a path, if there are any.
func buildQuery(path string, params url.Values) string {
	if len(params) == 0 {
		return path
	}
	return path + "?" + params.Encode()
}

// setQueryParam adds an optional query parameter, skipping nil values.
func setQueryParam[T any](params url.Values, key string, value *T) {
	if value == nil {
		return
	}

	params.Set(key, fmt.Sprintf("%v", *value))
}

// apiPath builds the path for an endpoint: a service prefix, an action, and the
// path parameters that follow. A nil parameter is skipped, which is how the
// optional trailing segments are omitted.
//
// Segments come from caller input such as domain and subdomain names, so each is
// escaped. A segment that is a relative reference cannot be escaped away, and is
// rejected in newRequest instead.
func apiPath(prefix string, action string, segments ...any) string {
	path := "/" + prefix + "/" + action

	for _, v := range segments {
		// The optional segments are passed as pointers so that nil means "omit".
		switch p := v.(type) {
		case nil:
			continue
		case *int64:
			if p == nil {
				continue
			}
			v = *p
		case *string:
			if p == nil {
				continue
			}
			v = *p
		}

		path += "/" + escapePathSegment(fmt.Sprintf("%v", v))
	}

	return path
}

// escapePathSegment escapes a value for use as a single path segment, so that a
// "/" in a domain or subdomain becomes "%2F" and stays inside its own segment.
//
// "*" is left as-is: it names a wildcard DNS record and cannot act as a
// separator, so escaping it would break wildcard lookups for no gain.
func escapePathSegment(segment string) string {
	return strings.ReplaceAll(url.PathEscape(segment), "%2A", "*")
}

// pathHasDotSegment reports whether any segment of path is "." or "..".
//
// Escaping cannot neutralise these, since url.PathEscape leaves a dot alone: a
// domain of ".." gives "/dns/delete/../5", which anything resolving the path
// before routing it reads as "/dns/5". Neither is a valid domain, subdomain, tld
// or id, so newRequest refuses the request. Checked on the assembled path, so
// every endpoint is covered by the one guard.
func pathHasDotSegment(path string) bool {
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}
