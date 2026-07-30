package porkbun

import (
	"context"
	"errors"
	"strings"
)

// DNSService provides methods to interact with the DNS record management API.
type DNSService struct {
	client *Client // Client used to communicate with the API
}

// getRecordsRequest represents the request structure for retrieving DNS records.
type getRecordsRequest struct {
	baseRequest // Embeds the baseRequest to include API credentials
}

// GetRecordsResponse represents the response structure for retrieving DNS records.
type GetRecordsResponse struct {
	BaseResponse
	Cloudflare CloudflareStatus `json:"cloudflare"` // Whether the Cloudflare proxy is enabled, empty when the API omits it
	Records    []DNSRecord      `json:"records"`    // List of DNS records
}

// createRecordRequest represents the request structure for creating a DNS record.
type createRecordRequest struct {
	baseRequest
	dryRunnable
	*DNSRecord // Embeds the DNSRecord to include DNS record details
}

// CreateRecordResponse represents the response structure for creating a DNS record.
// When the request ran with WithDryRun, DryRun is true and no record was created,
// so ID is zero.
type CreateRecordResponse struct {
	BaseResponse
	DryRunResult
	// ID of the newly created DNS record. FlexInt64, because the API returns it
	// as a JSON string on some endpoints and a number on others.
	ID FlexInt64 `json:"id"`
}

// editRecordRequest represents the request structure for editing a DNS record.
type editRecordRequest struct {
	baseRequest
	dryRunnable
	*EditRecord // Embeds the EditRecord to include DNS record details to be edited
}

// editRecordTypeRequest represents the request structure for editing DNS records by type.
type editRecordTypeRequest struct {
	baseRequest
	dryRunnable
	*EditTypeRecord // Embeds the EditTypeRecord to include DNS record details to be edited
}

// EditRecordResponse represents the response structure for editing a DNS record.
// When the request ran with WithDryRun, DryRun is true and nothing was changed.
type EditRecordResponse struct {
	BaseResponse
	DryRunResult
}

// deleteRecordRequest represents the request structure for deleting a DNS record.
type deleteRecordRequest struct {
	baseRequest
	dryRunnable
}

// DeleteRecordResponse represents the response structure for deleting a DNS record.
// When the request ran with WithDryRun, DryRun is true and nothing was deleted.
type DeleteRecordResponse struct {
	BaseResponse
	DryRunResult
}

// DNSRecordType represents a DNS record type as a string enum.
type DNSRecordType string

// Enum values for DNSRecordType
const (
	A     DNSRecordType = "A"
	MX    DNSRecordType = "MX"
	CNAME DNSRecordType = "CNAME"
	ALIAS DNSRecordType = "ALIAS"
	TXT   DNSRecordType = "TXT"
	NS    DNSRecordType = "NS"
	AAAA  DNSRecordType = "AAAA"
	SRV   DNSRecordType = "SRV"
	TLSA  DNSRecordType = "TLSA"
	CAA   DNSRecordType = "CAA"
	SSHFP DNSRecordType = "SSHFP"
	HTTPS DNSRecordType = "HTTPS"
	SVCB  DNSRecordType = "SVCB"
)

// IsValid checks if the DNSRecordType is valid.
func (rt DNSRecordType) IsValid() bool {
	switch rt {
	case A, MX, CNAME, ALIAS, TXT, NS, AAAA, SRV, TLSA, CAA, SSHFP, HTTPS, SVCB:
		return true
	}
	return false
}

// String returns the string representation of the DNSRecordType.
func (rt DNSRecordType) String() string {
	return string(rt)
}

// EditRecord represents the details required to edit a DNS record.
type EditRecord struct {
	Name    string        `json:"name"`            // Subdomain name for the DNS record
	Type    DNSRecordType `json:"type"`            // DNS record type
	Content string        `json:"content"`         // DNS record content
	TTL     int64         `json:"ttl,omitempty"`   // Time to live in seconds, zero for the account minimum
	Prio    int64         `json:"prio,omitempty"`  // Priority for MX and SRV records, zero otherwise
	Notes   *string       `json:"notes,omitempty"` // Notes. Pass an empty string to clear them, or nil to leave unchanged
}

// EditTypeRecord represents the details required to edit DNS records by type.
type EditTypeRecord struct {
	Content string  `json:"content"`         // DNS record content
	TTL     int64   `json:"ttl,omitempty"`   // Time to live in seconds, zero for the account minimum
	Prio    int64   `json:"prio,omitempty"`  // Priority for MX and SRV records, zero otherwise
	Notes   *string `json:"notes,omitempty"` // Notes. Pass an empty string to clear them, or nil to leave unchanged
}

// DNSRecord represents a DNS record in the system.
//
// The numeric fields are FlexInt64 because the API sends them as quoted strings
// on retrieval and accepts them as numbers on write, and prio arrives as null on
// record types that have no priority.
type DNSRecord struct {
	ID      FlexInt64     `json:"id,omitempty"`    // DNS record ID, zero for a record that does not exist yet
	Name    string        `json:"name"`            // Subdomain name for the DNS record
	Type    DNSRecordType `json:"type"`            // DNS record type
	Content string        `json:"content"`         // DNS record content
	TTL     FlexInt64     `json:"ttl,omitempty"`   // Time to live in seconds, zero for the account minimum
	Prio    FlexInt64     `json:"prio,omitempty"`  // Priority for MX and SRV records, zero otherwise
	Notes   string        `json:"notes,omitempty"` // Additional notes (optional)
}

// unqualifyName turns the fully qualified name a retrieved record carries into
// the bare subdomain the create and edit endpoints expect.
//
// The two directions disagree: retrieval reports "www.example.com", while the
// writes take the subdomain alone and prepend the domain themselves, so a
// retrieved name passed straight back would address
// www.example.com.example.com. A bare name is left alone, and the domain itself
// becomes "", which addresses the root record. Matched case-insensitively,
// because DNS names are.
func unqualifyName(name string, domain string) string {
	// DNS tooling writes absolute names with the root dot; the API never does.
	name = strings.TrimSuffix(name, ".")

	if strings.EqualFold(name, domain) {
		return ""
	}

	suffix := "." + domain
	if len(name) > len(suffix) && strings.EqualFold(name[len(name)-len(suffix):], suffix) {
		return name[:len(name)-len(suffix)]
	}

	return name
}

// unqualifySubdomain applies unqualifyName to an optional path segment, so the
// byType endpoints address the same records as the create and edit bodies. A nil
// subdomain stays nil, which leaves the segment off for the root domain.
func unqualifySubdomain(subdomain *string, domain string) *string {
	if subdomain == nil {
		return nil
	}

	bare := unqualifyName(*subdomain, domain)
	if bare == "" {
		// The root record, which the API addresses by omitting the segment. Both
		// that and the empty trailing segment are accepted, so normalise to one.
		return nil
	}

	return &bare
}

// ListRecords returns every DNS record for a domain.
func (s *DNSService) ListRecords(ctx context.Context, domain string) (*GetRecordsResponse, error) {
	return s.getRecords(ctx, apiPath("dns", "retrieve", domain))
}

// GetRecord returns the single DNS record with the given id. The record is still
// delivered in Records, which holds one entry.
func (s *DNSService) GetRecord(ctx context.Context, domain string, recordID int64) (*GetRecordsResponse, error) {
	return s.getRecords(ctx, apiPath("dns", "retrieve", domain, recordID))
}

// GetRecords retrieves DNS records for a domain, optionally filtered by record ID.
//
// Deprecated: use ListRecords, or GetRecord for a single id. A pointer that
// selects between two unrelated operations reads poorly at the call site.
func (s *DNSService) GetRecords(ctx context.Context, domain string, recordID *int64) (*GetRecordsResponse, error) {
	return s.getRecords(ctx, apiPath("dns", "retrieve", domain, recordID))
}

// getRecords posts to a retrieval path, which is the same for all three forms.
func (s *DNSService) getRecords(ctx context.Context, path string) (*GetRecordsResponse, error) {
	request := &getRecordsRequest{}
	response := &GetRecordsResponse{}

	_, err := s.client.post(ctx, path, request, response)
	return response, err
}

// GetRecordsByType retrieves DNS records for a domain by record type and subdomain.
//
// subdomain is the subdomain alone, and a fully qualified name carried over from
// GetRecords loses the domain again, as it does for CreateRecord.
func (s *DNSService) GetRecordsByType(ctx context.Context, domain string, recordType DNSRecordType, subdomain *string) (*GetRecordsResponse, error) {
	path := apiPath("dns", "retrieveByNameType", domain, recordType, unqualifySubdomain(subdomain, domain))

	request := &getRecordsRequest{}
	response := &GetRecordsResponse{}

	_, err := s.client.post(ctx, path, request, response)
	return response, err
}

// CreateRecord creates a new DNS record for a domain.
func (s *DNSService) CreateRecord(ctx context.Context, domain string, record *DNSRecord, opts ...RequestOption) (*CreateRecordResponse, error) {
	if record == nil {
		return &CreateRecordResponse{}, errors.New("porkbun: CreateRecord requires a record, at least Type and Content")
	}

	path := apiPath("dns", "create", domain)

	// The API allocates the id, and qualifies the name. A record read back from
	// GetRecords carries both already, so it can be passed straight back here.
	create := *record
	create.ID = 0
	create.Name = unqualifyName(create.Name, domain)

	request := &createRecordRequest{
		DNSRecord: &create,
	}
	response := &CreateRecordResponse{}

	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}

// EditRecord edits an existing DNS record for a domain by record ID.
func (s *DNSService) EditRecord(ctx context.Context, domain string, recordID int64, record *EditRecord, opts ...RequestOption) (*EditRecordResponse, error) {
	if record == nil {
		return &EditRecordResponse{}, errors.New("porkbun: EditRecord requires a record, at least Type and Content")
	}

	path := apiPath("dns", "edit", domain, recordID)

	// The API qualifies the name itself, so a name carried over from GetRecords
	// has to lose the domain again.
	edit := *record
	edit.Name = unqualifyName(edit.Name, domain)

	request := &editRecordRequest{
		EditRecord: &edit,
	}
	response := &EditRecordResponse{}

	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}

// EditRecordByType edits all DNS records for a domain that match a particular type and subdomain.
//
// subdomain is the subdomain alone, and a fully qualified name carried over from
// GetRecords loses the domain again, as it does for EditRecord.
func (s *DNSService) EditRecordByType(ctx context.Context, domain string, recordType DNSRecordType, subdomain *string, record *EditTypeRecord, opts ...RequestOption) (*EditRecordResponse, error) {
	if record == nil {
		return &EditRecordResponse{}, errors.New("porkbun: EditRecordByType requires a record, at least Content")
	}

	path := apiPath("dns", "editByNameType", domain, recordType, unqualifySubdomain(subdomain, domain))

	// Copied, so the caller's record is neither read while it is being encoded nor
	// written to.
	edit := *record
	request := &editRecordTypeRequest{
		EditTypeRecord: &edit,
	}
	response := &EditRecordResponse{}

	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}

// DeleteRecord deletes a specific DNS record for a domain by record ID.
func (s *DNSService) DeleteRecord(ctx context.Context, domain string, recordID int64, opts ...RequestOption) (*DeleteRecordResponse, error) {
	path := apiPath("dns", "delete", domain, recordID)

	request := &deleteRecordRequest{}
	response := &DeleteRecordResponse{}

	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}

// DeleteRecordByType deletes all DNS records for a domain that match a particular type and (optional) subdomain.
//
// subdomain is the subdomain alone, and a fully qualified name carried over from
// GetRecords loses the domain again: passing "www.example.com" through
// unchanged would match nothing and delete nothing.
func (s *DNSService) DeleteRecordByType(ctx context.Context, domain string, recordType DNSRecordType, subdomain *string, opts ...RequestOption) (*DeleteRecordResponse, error) {
	path := apiPath("dns", "deleteByNameType", domain, recordType, unqualifySubdomain(subdomain, domain))

	request := &deleteRecordRequest{}
	response := &DeleteRecordResponse{}

	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}
