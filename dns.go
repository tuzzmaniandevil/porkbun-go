package porkbun

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// DnsService provides methods to interact with the DNS record management API.
type DnsService struct {
	client *Client // Client used to communicate with the API
}

// dnsPath constructs the path for DNS-related API endpoints.
func dnsPath(action string, a ...any) string {
	path := fmt.Sprintf("/dns/%v", action)
	buildPathSegments(&path, a...)
	return path
}

// GetRecordsRequest represents the request structure for retrieving DNS records.
type GetRecordsRequest struct {
	BaseRequest // Embeds the BaseRequest to include API credentials
}

// GetRecordsResponse represents the response structure for retrieving DNS records.
type GetRecordsResponse struct {
	BaseResponse
	Records []DnsRecord `json:"records"` // List of DNS records
}

// CreateRecordRequest represents the request structure for creating a DNS record.
type CreateRecordRequest struct {
	BaseRequest
	*DnsRecord // Embeds the DnsRecord to include DNS record details
}

// CreateRecordResponse represents the response structure for creating a DNS record.
type CreateRecordResponse struct {
	BaseResponse
	ID int64 `json:"id"` // ID of the newly created DNS record
}

// EditRecordRequest represents the request structure for editing a DNS record.
type EditRecordRequest struct {
	BaseRequest
	*EditRecord // Embeds the EditRecord to include DNS record details to be edited
}

// EditRecordTypeRequest represents the request structure for editing DNS records by type.
type EditRecordTypeRequest struct {
	BaseRequest
	*EditTypeRecord // Embeds the EditTypeRecord to include DNS record details to be edited
}

// EditRecordResponse represents the response structure for editing a DNS record.
type EditRecordResponse struct {
	BaseResponse
}

// DeleteRecordRequest represents the request structure for deleting a DNS record.
type DeleteRecordRequest struct {
	BaseRequest
}

// DeleteRecordResponse represents the response structure for deleting a DNS record.
type DeleteRecordResponse struct {
	BaseResponse
}

// DnsRecordType represents a DNS record type as a string enum.
type DnsRecordType string

// Enum values for DnsRecordType
const (
	A     DnsRecordType = "A"
	MX    DnsRecordType = "MX"
	CNAME DnsRecordType = "CNAME"
	ALIAS DnsRecordType = "ALIAS"
	TXT   DnsRecordType = "TXT"
	NS    DnsRecordType = "NS"
	AAAA  DnsRecordType = "AAAA"
	SRV   DnsRecordType = "SRV"
	TLSA  DnsRecordType = "TLSA"
	CAA   DnsRecordType = "CAA"
	HTTPS DnsRecordType = "HTTPS"
	SVCB  DnsRecordType = "SVCB"
)

// IsValid checks if the DnsRecordType is valid.
func (rt DnsRecordType) IsValid() bool {
	switch rt {
	case A, MX, CNAME, ALIAS, TXT, NS, AAAA, SRV, TLSA, CAA, HTTPS, SVCB:
		return true
	}
	return false
}

// String returns the string representation of the DnsRecordType.
func (rt DnsRecordType) String() string {
	return string(rt)
}

// ToPtr returns a pointer to the string representation of the DnsRecordType.
func (rt DnsRecordType) ToPtr() *string {
	s := string(rt)
	return &s
}

// EditRecord represents the details required to edit a DNS record.
type EditRecord struct {
	Name    string        `json:"name"`           // Subdomain name for the DNS record
	Type    DnsRecordType `json:"type"`           // DNS record type
	Content string        `json:"content"`        // DNS record content
	TTL     string        `json:"ttl,omitempty"`  // Time to live (TTL) for the DNS record
	Prio    string        `json:"prio,omitempty"` // Priority for the DNS record (if applicable)
}

// EditTypeRecord represents the details required to edit DNS records by type.
type EditTypeRecord struct {
	Content string `json:"content"`        // DNS record content
	TTL     string `json:"ttl,omitempty"`  // Time to live (TTL) for the DNS record
	Prio    string `json:"prio,omitempty"` // Priority for the DNS record (if applicable)
}

// DnsRecord represents a DNS record in the system.
type DnsRecord struct {
	ID      *int64        `json:"id,omitempty"`    // DNS record ID (optional for new records)
	Name    string        `json:"name"`            // Subdomain name for the DNS record
	Type    DnsRecordType `json:"type"`            // DNS record type
	Content string        `json:"content"`         // DNS record content
	TTL     string        `json:"ttl,omitempty"`   // Time to live (TTL) for the DNS record
	Prio    string        `json:"prio,omitempty"`  // Priority for the DNS record (if applicable)
	Notes   string        `json:"notes,omitempty"` // Additional notes (optional)
}

// UnmarshalJSON handles the custom unmarshalling of the DnsRecord struct.
func (d *DnsRecord) UnmarshalJSON(data []byte) error {
	// Define a temporary struct to capture the raw values
	type Alias DnsRecord
	aux := &struct {
		ID *string `json:"id"`
		*Alias
	}{
		Alias: (*Alias)(d),
	}

	// Unmarshal into the temporary struct
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Convert the ID from string to int64 if present
	if aux.ID != nil {
		id, err := strconv.ParseInt(*aux.ID, 10, 64)
		if err != nil {
			return err
		}
		d.ID = &id
	}

	return nil
}

// GetRecords retrieves DNS records for a domain, optionally filtered by record ID.
func (s *DnsService) GetRecords(ctx context.Context, domain string, recordId *int64) (*GetRecordsResponse, error) {
	path := dnsPath("retrieve", domain, recordId)

	request := &GetRecordsRequest{}
	response := &GetRecordsResponse{}

	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}

// GetRecordsByType retrieves DNS records for a domain by record type and subdomain.
func (s *DnsService) GetRecordsByType(ctx context.Context, domain string, recordType DnsRecordType, subdomain *string) (*GetRecordsResponse, error) {
	path := dnsPath("retrieveByNameType", domain, recordType, subdomain)

	request := &GetRecordsRequest{}
	response := &GetRecordsResponse{}

	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}

// CreateRecord creates a new DNS record for a domain.
func (s *DnsService) CreateRecord(ctx context.Context, domain string, record *DnsRecord) (*CreateRecordResponse, error) {
	path := dnsPath("create", domain)

	request := &CreateRecordRequest{
		DnsRecord: record,
	}
	response := &CreateRecordResponse{}

	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}

// EditRecord edits an existing DNS record for a domain by record ID.
func (s *DnsService) EditRecord(ctx context.Context, domain string, recordId int64, record *EditRecord) (*EditRecordResponse, error) {
	path := dnsPath("edit", domain, recordId)

	request := &EditRecordRequest{
		EditRecord: record,
	}
	response := &EditRecordResponse{}

	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}

// EditRecordByType edits all DNS records for a domain that match a particular type and subdomain.
func (s *DnsService) EditRecordByType(ctx context.Context, domain string, recordType DnsRecordType, subdomain *string, record *EditTypeRecord) (*EditRecordResponse, error) {
	path := dnsPath("editByNameType", domain, recordType, subdomain)

	request := &EditRecordTypeRequest{
		EditTypeRecord: record,
	}
	response := &EditRecordResponse{}

	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}

// DeleteRecord deletes a specific DNS record for a domain by record ID.
func (s *DnsService) DeleteRecord(ctx context.Context, domain string, recordId int64) (*DeleteRecordResponse, error) {
	path := dnsPath("delete", domain, recordId)

	request := &DeleteRecordRequest{}
	response := &DeleteRecordResponse{}

	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}

// DeleteRecordByType deletes all DNS records for a domain that match a particular type and (optional) subdomain.
func (s *DnsService) DeleteRecordByType(ctx context.Context, domain string, recordType DnsRecordType, subdomain *string) (*DeleteRecordResponse, error) {
	path := dnsPath("deleteByNameType", domain, recordType, subdomain)

	request := &DeleteRecordRequest{}
	response := &DeleteRecordResponse{}

	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}

// Interface guards ensure that the required interfaces are implemented by the request types.
var (
	_ ApiKeyAcceptor = (*GetRecordsRequest)(nil)
	_ ApiKeyAcceptor = (*CreateRecordRequest)(nil)
	_ ApiKeyAcceptor = (*EditRecordRequest)(nil)
	_ ApiKeyAcceptor = (*EditRecordTypeRequest)(nil)
	_ ApiKeyAcceptor = (*DeleteRecordRequest)(nil)
)
