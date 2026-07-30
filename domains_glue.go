package porkbun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// GlueIPs holds the IP addresses associated with a glue record.
type GlueIPs struct {
	V4 []string `json:"v4"` // IPv4 addresses
	V6 []string `json:"v6"` // IPv6 addresses
}

// GlueRecord represents a glue record (host object) registered under a domain.
// The API returns each record as a [hostname, ips] tuple.
type GlueRecord struct {
	Host string  // Full hostname, e.g. "ns1.example.com"
	IPs  GlueIPs // IP addresses associated with the host
}

// UnmarshalJSON decodes the two-element [hostname, ips] tuple the API returns.
func (g *GlueRecord) UnmarshalJSON(data []byte) error {
	var tuple []json.RawMessage
	if err := json.Unmarshal(data, &tuple); err != nil {
		return err
	}

	if len(tuple) != 2 {
		return fmt.Errorf("invalid glue record format: expected 2 elements, got %d", len(tuple))
	}

	if err := json.Unmarshal(tuple[0], &g.Host); err != nil {
		return err
	}

	return json.Unmarshal(tuple[1], &g.IPs)
}

// getGlueRecordsRequest represents the request structure for retrieving glue records.
type getGlueRecordsRequest struct {
	baseRequest
}

// MarshalJSON re-encodes the record as the two-element [hostname, ips] tuple the
// API uses, so a decoded response round-trips. Without it the Go field names
// would reach the wire and UnmarshalJSON would reject its own output.
func (g GlueRecord) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{g.Host, g.IPs})
}

// GetGlueRecordsResponse represents the response structure for retrieving glue records.
type GetGlueRecordsResponse struct {
	BaseResponse
	Hosts []GlueRecord `json:"hosts"` // Glue records registered under the domain
}

// glueRecordRequest represents the request structure for creating or updating a glue record.
type glueRecordRequest struct {
	baseRequest
	IPs []string `json:"ips,omitempty"` // IP addresses (IPv4 and/or IPv6) to associate with the host record
}

// GlueRecordResponse represents the response structure for a glue record write.
type GlueRecordResponse struct {
	BaseResponse
}

// GetGlueRecords retrieves all glue records registered under the domain.
func (s *DomainsService) GetGlueRecords(ctx context.Context, domain string) (*GetGlueRecordsResponse, error) {
	path := apiPath("domain", "getGlue", domain)

	request := &getGlueRecordsRequest{}
	response := &GetGlueRecordsResponse{}

	_, err := s.client.post(ctx, path, request, response)
	return response, err
}

// CreateGlueRecord creates a glue record for a nameserver hosted under the domain.
// subdomain is the host portion only, e.g. "ns1" for ns1.example.com.
func (s *DomainsService) CreateGlueRecord(ctx context.Context, domain string, subdomain string, ips []string, opts ...RequestOption) (*GlueRecordResponse, error) {
	if len(ips) == 0 {
		return &GlueRecordResponse{}, errors.New("porkbun: CreateGlueRecord requires at least one IP address")
	}

	return s.writeGlueRecord(ctx, "createGlue", domain, subdomain, ips, opts...)
}

// UpdateGlueRecord replaces all IP addresses on an existing glue record.
// subdomain is the host portion only, e.g. "ns1" for ns1.example.com.
func (s *DomainsService) UpdateGlueRecord(ctx context.Context, domain string, subdomain string, ips []string, opts ...RequestOption) (*GlueRecordResponse, error) {
	if len(ips) == 0 {
		return &GlueRecordResponse{}, errors.New("porkbun: UpdateGlueRecord requires at least one IP address, use DeleteGlueRecord to remove the record")
	}

	return s.writeGlueRecord(ctx, "updateGlue", domain, subdomain, ips, opts...)
}

// DeleteGlueRecord deletes the glue record for a subdomain.
func (s *DomainsService) DeleteGlueRecord(ctx context.Context, domain string, subdomain string, opts ...RequestOption) (*GlueRecordResponse, error) {
	return s.writeGlueRecord(ctx, "deleteGlue", domain, subdomain, nil, opts...)
}

// writeGlueRecord performs a glue record write, which is the same shape for
// create, update and delete.
func (s *DomainsService) writeGlueRecord(ctx context.Context, action string, domain string, subdomain string, ips []string, opts ...RequestOption) (*GlueRecordResponse, error) {
	// getGlue reports Host as the full hostname while the writes take the host
	// portion alone, so a name carried over from GetGlueRecords loses the domain
	// again, as it does for the DNS writes.
	path := apiPath("domain", action, domain, unqualifyName(subdomain, domain))

	request := &glueRecordRequest{IPs: ips}
	response := &GlueRecordResponse{}

	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}

var (
	_ json.Unmarshaler = (*GlueRecord)(nil)
	_ json.Marshaler   = GlueRecord{}
)
