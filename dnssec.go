package porkbun

import (
	"context"
)

// DnssecAlgorithm represents the algorithm used for DNSSEC.
type DnssecAlgorithm string

const (
	DnssecAlgorithmRsaMd5        DnssecAlgorithm = "1"
	DnssecAlgorithmDsaSha1       DnssecAlgorithm = "3"
	DnssecAlgorithmRsaSha1       DnssecAlgorithm = "5"
	DnssecAlgorithmDsaNsec3Sha1  DnssecAlgorithm = "6"
	DnssecAlgorithmRsaSha256     DnssecAlgorithm = "8"
	DnssecAlgorithmRsaSha512     DnssecAlgorithm = "10"
	DnssecAlgorithmGostR34111994 DnssecAlgorithm = "12"
	DnssecAlgorithmEcdsaSha256   DnssecAlgorithm = "13"
	DnssecAlgorithmEcdsaSha384   DnssecAlgorithm = "14"
	DnssecAlgorithmEd25519       DnssecAlgorithm = "15"
	DnssecAlgorithmEd448         DnssecAlgorithm = "16"
)

// IsValid checks if the DnssecAlgorithm is valid.
func (da DnssecAlgorithm) IsValid() bool {
	switch da {
	case DnssecAlgorithmRsaMd5,
		DnssecAlgorithmDsaSha1,
		DnssecAlgorithmRsaSha1,
		DnssecAlgorithmDsaNsec3Sha1,
		DnssecAlgorithmRsaSha256,
		DnssecAlgorithmRsaSha512,
		DnssecAlgorithmGostR34111994,
		DnssecAlgorithmEcdsaSha256,
		DnssecAlgorithmEcdsaSha384,
		DnssecAlgorithmEd25519,
		DnssecAlgorithmEd448:
		return true
	default:
		return false
	}
}

// DnssecDigestType represents the digest type used for DNSSEC.
type DnssecDigestType string

const (
	DnssecDigestTypeSha1          DnssecDigestType = "1"
	DnssecDigestTypeSha256        DnssecDigestType = "2"
	DnssecDigestTypeGostR34111994 DnssecDigestType = "3"
	DnssecDigestTypeSha384        DnssecDigestType = "4"
)

// IsValid checks if the DnssecDigestType is valid.
func (dt DnssecDigestType) IsValid() bool {
	switch dt {
	case DnssecDigestTypeSha1,
		DnssecDigestTypeSha256,
		DnssecDigestTypeGostR34111994,
		DnssecDigestTypeSha384:
		return true
	default:
		return false
	}
}

// GetDnssecRecordsRequest represents the request structure for retrieving DNSSEC records.
type GetDnssecRecordsRequest struct {
	BaseRequest
}

// GetDnssecRecordsResponse represents the request structure for retrieving DNSSEC records.
type GetDnssecRecordsResponse struct {
	BaseResponse
	Records map[string]DnssecRecordData `json:"records"`
}

// CreateDnssecRecordRequest represents the request structure for creating a DNSSEC record.
type CreateDnssecRecordRequest struct {
	BaseRequest
	*DnssecRecordData // Embeds the DnssecRecordData to include DNSSEC record details
}

// CreateDnssecRecordResponse represents the response structure for creating a DNSSEC record.
type CreateDnssecRecordResponse struct {
	BaseResponse
}

// DeleteDnssecRecordRequest represents the request structure for deleting a DNSSEC record.
type DeleteDnssecRecordRequest struct {
	BaseRequest
}

// DeleteDnssecRecordResponse represents the response structure for deleting a DNSSEC record.
type DeleteDnssecRecordResponse struct {
	BaseResponse
}

// DnssecRecordData represents a DNSSEC record for a domain.
type DnssecRecordData struct {
	KeyTag          string           `json:"keyTag"`
	Alg             DnssecAlgorithm  `json:"alg"`
	DigestType      DnssecDigestType `json:"digestType"`
	Digest          string           `json:"digest"`
	MaxSigLife      string           `json:"maxSigLife,omitempty"`
	KeyDataFlags    *string          `json:"keyDataFlags,omitempty"`
	KeyDataProtocol *string          `json:"keyDataProtocol,omitempty"`
	KeyDataAlgo     *DnssecAlgorithm `json:"keyDataAlgo,omitempty"`
	KeyDataPubKey   *string          `json:"keyDataPubKey,omitempty"`
}

// GetDnssecRecords retrieves the DNSSEC records for a specified domain.
func (s *DnsService) GetDnssecRecords(ctx context.Context, domain string) (*GetDnssecRecordsResponse, error) {
	path := dnsPath("getDnssecRecords", domain)

	request := &GetDnssecRecordsRequest{}
	response := &GetDnssecRecordsResponse{}

	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}

// CreateDnssecRecord creates a new DNSSEC record for a specified domain.
func (s *DnsService) CreateDnssecRecord(ctx context.Context, domain string, record *DnssecRecordData) (*CreateDnssecRecordResponse, error) {
	path := dnsPath("createDnssecRecord", domain)

	request := &CreateDnssecRecordRequest{
		DnssecRecordData: record,
	}
	response := &CreateDnssecRecordResponse{}

	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}

// DeleteDnssecRecord deletes a DNSSEC record for a specified domain.
func (s *DnsService) DeleteDnssecRecord(ctx context.Context, domain string, recordID string) (*DeleteDnssecRecordResponse, error) {
	path := dnsPath("deleteDnssecRecord", domain, recordID)

	request := &DeleteDnssecRecordRequest{}
	response := &DeleteDnssecRecordResponse{}

	resp, err := s.client.post(ctx, path, request, response)
	if err != nil {
		return response, err
	}

	response.HTTPResponse = resp
	return response, err
}
