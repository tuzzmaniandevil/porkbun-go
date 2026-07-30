package porkbun

import (
	"context"
	"encoding/json"
	"errors"
)

// DNSSECAlgorithm is a DS record signing algorithm, as its IANA DNSSEC Algorithm
// Numbers registry value. The API types the field as a free-form string and
// documents no enum, so the constants below are this package's model of the
// registry rather than a list the API enforces.
type DNSSECAlgorithm string

// Algorithm numbers from the IANA DNSSEC Algorithm Numbers registry.
const (
	DNSSECAlgorithmRSAMD5          DNSSECAlgorithm = "1"  // RSA/MD5, deprecated and not usable for signing
	DNSSECAlgorithmDSASHA1         DNSSECAlgorithm = "3"  // DSA/SHA-1
	DNSSECAlgorithmRSASHA1         DNSSECAlgorithm = "5"  // RSA/SHA-1
	DNSSECAlgorithmDSANSEC3SHA1    DNSSECAlgorithm = "6"  // DSA-NSEC3-SHA1
	DNSSECAlgorithmRSASHA1NSEC3    DNSSECAlgorithm = "7"  // RSASHA1-NSEC3-SHA1
	DNSSECAlgorithmRSASHA256       DNSSECAlgorithm = "8"  // RSA/SHA-256
	DNSSECAlgorithmRSASHA512       DNSSECAlgorithm = "10" // RSA/SHA-512
	DNSSECAlgorithmECCGOST         DNSSECAlgorithm = "12" // ECC-GOST, a GOST R 34.10-2001 signature
	DNSSECAlgorithmECDSAP256SHA256 DNSSECAlgorithm = "13" // ECDSA Curve P-256 with SHA-256
	DNSSECAlgorithmECDSAP384SHA384 DNSSECAlgorithm = "14" // ECDSA Curve P-384 with SHA-384
	DNSSECAlgorithmED25519         DNSSECAlgorithm = "15" // Ed25519
	DNSSECAlgorithmED448           DNSSECAlgorithm = "16" // Ed448
)

// IsValid reports whether the algorithm is one of the registry values above.
//
// It is a convenience for a caller inspecting a retrieved record; the SDK does
// not call it, and the registry is the authority on what a registry will accept.
func (da DNSSECAlgorithm) IsValid() bool {
	switch da {
	case DNSSECAlgorithmRSAMD5,
		DNSSECAlgorithmDSASHA1,
		DNSSECAlgorithmRSASHA1,
		DNSSECAlgorithmDSANSEC3SHA1,
		DNSSECAlgorithmRSASHA1NSEC3,
		DNSSECAlgorithmRSASHA256,
		DNSSECAlgorithmRSASHA512,
		DNSSECAlgorithmECCGOST,
		DNSSECAlgorithmECDSAP256SHA256,
		DNSSECAlgorithmECDSAP384SHA384,
		DNSSECAlgorithmED25519,
		DNSSECAlgorithmED448:
		return true
	default:
		return false
	}
}

// DNSSECDigestType is the hash used for a DS record digest, as its IANA DS RR
// Type Digest Algorithms registry value. As with DNSSECAlgorithm, the API types
// the field as a free-form string and documents no enum.
type DNSSECDigestType string

// Digest algorithms from the IANA DS RR Type Digest Algorithms registry.
const (
	DNSSECDigestTypeSHA1        DNSSECDigestType = "1" // SHA-1
	DNSSECDigestTypeSHA256      DNSSECDigestType = "2" // SHA-256
	DNSSECDigestTypeGOSTR341194 DNSSECDigestType = "3" // GOST R 34.11-94
	DNSSECDigestTypeSHA384      DNSSECDigestType = "4" // SHA-384
)

// IsValid reports whether the digest type is one of the registry values above.
func (dt DNSSECDigestType) IsValid() bool {
	switch dt {
	case DNSSECDigestTypeSHA1,
		DNSSECDigestTypeSHA256,
		DNSSECDigestTypeGOSTR341194,
		DNSSECDigestTypeSHA384:
		return true
	default:
		return false
	}
}

// getDnssecRecordsRequest represents the request structure for retrieving DNSSEC records.
type getDnssecRecordsRequest struct {
	baseRequest
}

// DNSSECRecords maps a key tag to its DNSSEC record. A domain with DNSSEC
// disabled decodes as a nil map.
type DNSSECRecords map[string]DNSSECRecord

// UnmarshalJSON implements custom unmarshalling logic for DNSSECRecords.
func (d *DNSSECRecords) UnmarshalJSON(data []byte) error {
	return unmarshalFlexMap(data, d, "dnssec records")
}

var _ json.Unmarshaler = (*DNSSECRecords)(nil)

// GetDnssecRecordsResponse represents the request structure for retrieving DNSSEC records.
type GetDnssecRecordsResponse struct {
	BaseResponse
	Records DNSSECRecords `json:"records"` // DNSSEC records keyed by key tag
}

// createDnssecRecordRequest represents the request structure for creating a DNSSEC record.
type createDnssecRecordRequest struct {
	baseRequest
	*DNSSECRecord // Embeds the DNSSECRecord to include DNSSEC record details
}

// CreateDnssecRecordResponse represents the response structure for creating a DNSSEC record.
type CreateDnssecRecordResponse struct {
	BaseResponse
}

// deleteDnssecRecordRequest represents the request structure for deleting a DNSSEC record.
type deleteDnssecRecordRequest struct {
	baseRequest
}

// DeleteDnssecRecordResponse represents the response structure for deleting a DNSSEC record.
type DeleteDnssecRecordResponse struct {
	BaseResponse
}

// DNSSECRecord is a DNSSEC DS or key record registered for a domain at its
// registry.
//
// The type serves both directions, and the API does not use every field in both.
// PubKey is reported only by GetDNSSECRecords and is dropped by
// CreateDNSSECRecord; MaxSigLife and the KeyData fields are accepted only by
// CreateDNSSECRecord and are always nil on a record read back.
//
// KeyTag, Alg, DigestType and Digest are the minimum a create needs. Which of
// the rest a registry accepts varies by registry.
type DNSSECRecord struct {
	KeyTag          string           `json:"keyTag"`                    // DNSSEC key tag
	Alg             DNSSECAlgorithm  `json:"alg"`                       // DS data algorithm number
	DigestType      DNSSECDigestType `json:"digestType"`                // Digest type number
	Digest          string           `json:"digest"`                    // Hex-encoded digest value
	PubKey          *string          `json:"pubKey,omitempty"`          // Public key, reported for key data records and never sent
	MaxSigLife      *string          `json:"maxSigLife,omitempty"`      // Maximum signature lifetime in seconds, registry-specific
	KeyDataFlags    *string          `json:"keyDataFlags,omitempty"`    // Key data flags, when submitting full key data
	KeyDataProtocol *string          `json:"keyDataProtocol,omitempty"` // Key data protocol
	KeyDataAlgo     *DNSSECAlgorithm `json:"keyDataAlgo,omitempty"`     // Key data algorithm
	KeyDataPubKey   *string          `json:"keyDataPubKey,omitempty"`   // Key data public key, base64
}

// GetDNSSECRecords retrieves the DNSSEC records for a specified domain.
func (s *DNSService) GetDNSSECRecords(ctx context.Context, domain string) (*GetDnssecRecordsResponse, error) {
	path := apiPath("dns", "getDnssecRecords", domain)

	request := &getDnssecRecordsRequest{}
	response := &GetDnssecRecordsResponse{}

	_, err := s.client.post(ctx, path, request, response)
	return response, err
}

// CreateDNSSECRecord creates a new DNSSEC record for a specified domain.
func (s *DNSService) CreateDNSSECRecord(ctx context.Context, domain string, record *DNSSECRecord, opts ...RequestOption) (*CreateDnssecRecordResponse, error) {
	if record == nil {
		return &CreateDnssecRecordResponse{}, errors.New("porkbun: CreateDNSSECRecord requires a record, at least KeyTag, Alg, DigestType and Digest")
	}

	path := apiPath("dns", "createDnssecRecord", domain)

	// Copied, so the caller's record is not read while it is being encoded.
	//
	// PubKey is reported only on retrieval and is not a field of the create body,
	// so it is dropped rather than carried in from a record read back by
	// GetDNSSECRecords. Submit key data through the KeyData fields.
	create := *record
	create.PubKey = nil
	request := &createDnssecRecordRequest{
		DNSSECRecord: &create,
	}
	response := &CreateDnssecRecordResponse{}

	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}

// DeleteDNSSECRecord deletes the DNSSEC record with the given key tag.
//
// Most registries delete every record matching the key data, not only the one
// carrying this key tag.
func (s *DNSService) DeleteDNSSECRecord(ctx context.Context, domain string, keyTag string, opts ...RequestOption) (*DeleteDnssecRecordResponse, error) {
	path := apiPath("dns", "deleteDnssecRecord", domain, keyTag)

	request := &deleteDnssecRecordRequest{}
	response := &DeleteDnssecRecordResponse{}

	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}
