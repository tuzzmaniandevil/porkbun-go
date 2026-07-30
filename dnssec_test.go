package porkbun

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDnssecAlgorithm_IsValid(t *testing.T) {
	tests := []struct {
		name string
		da   DNSSECAlgorithm
		want bool
	}{
		{"RSA/MD5", DNSSECAlgorithmRSAMD5, true},
		{"DSA/SHA-1", DNSSECAlgorithmDSASHA1, true},
		{"RSA/SHA-1", DNSSECAlgorithmRSASHA1, true},
		{"DSA-NSEC3-SHA1", DNSSECAlgorithmDSANSEC3SHA1, true},
		{"RSASHA1-NSEC3-SHA1", DNSSECAlgorithmRSASHA1NSEC3, true},
		{"RSA/SHA-256", DNSSECAlgorithmRSASHA256, true},
		{"RSA/SHA-512", DNSSECAlgorithmRSASHA512, true},
		{"GOST R 34.10-2001", DNSSECAlgorithmECCGOST, true},
		{"ECDSA/SHA-256", DNSSECAlgorithmECDSAP256SHA256, true},
		{"ECDSA/SHA-384", DNSSECAlgorithmECDSAP384SHA384, true},
		{"ED25519", DNSSECAlgorithmED25519, true},
		{"ED448", DNSSECAlgorithmED448, true},
		{"Invalid", "invalid", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.da.IsValid(), "IsValid()")
		})
	}
}

func TestDnssecDigestType_IsValid(t *testing.T) {
	tests := []struct {
		name string
		dt   DNSSECDigestType
		want bool
	}{
		{"SHA-1", DNSSECDigestTypeSHA1, true},
		{"SHA-256", DNSSECDigestTypeSHA256, true},
		{"GOST R 34.11-94", DNSSECDigestTypeGOSTR341194, true},
		{"SHA-384", DNSSECDigestTypeSHA384, true},
		{"Invalid", "invalid", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.dt.IsValid(), "IsValid()")
		})
	}
}

func TestDnsService_GetDnssecRecords_Success(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/getDnssecRecords/example.com", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/dns/getDnssecRecords/success.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)
		testCredentials(t, r)

		for k, values := range httpResponse.Header {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}

		w.WriteHeader(httpResponse.StatusCode)
		_, err := io.Copy(w, httpResponse.Body)

		require.NoError(t, err)
	})

	resp, err := client.DNS.GetDNSSECRecords(context.Background(), "example.com")

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.NotEmpty(t, resp.Records)

	record := resp.Records["64087"]
	assert.Equal(t, "64087", record.KeyTag)
	assert.Equal(t, DNSSECAlgorithmECDSAP256SHA256, record.Alg)
	assert.Equal(t, DNSSECDigestTypeSHA256, record.DigestType)
	assert.Equal(t, "15E445BD08128BDC213E25F1C8227DF4CB35186CAC701C1C335B2C406D5530DC", record.Digest)
}

func TestDnsService_CreateDnssecRecords_Success(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/createDnssecRecord/example.com", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/dns/createDnssecRecord/success.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)
		testCredentials(t, r)

		for k, values := range httpResponse.Header {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}

		w.WriteHeader(httpResponse.StatusCode)
		_, err := io.Copy(w, httpResponse.Body)

		require.NoError(t, err)
	})

	resp, err := client.DNS.CreateDNSSECRecord(context.Background(), "example.com", &DNSSECRecord{
		KeyTag:     "64087",
		Alg:        "13",
		DigestType: "2",
		Digest:     "15E445BD08128BDC213E25F1C8227DF4CB35186CAC701C1C335B2C406D5530DC",
	})

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

func TestDnsService_DeleteDnssecRecords_Success(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/deleteDnssecRecord/example.com/64087", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/dns/deleteDnssecRecord/success.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)
		testCredentials(t, r)

		for k, values := range httpResponse.Header {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}

		w.WriteHeader(httpResponse.StatusCode)
		_, err := io.Copy(w, httpResponse.Body)

		require.NoError(t, err)
	})

	resp, err := client.DNS.DeleteDNSSECRecord(context.Background(), "example.com", "64087")

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

// Key data records carry a public key alongside the DS fields.
func TestDns_GetDnssecRecords_KeyData(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/dns/getDnssecRecords/example.com", "POST", "/dns/getDnssecRecords/keydata.http")

	resp, err := client.DNS.GetDNSSECRecords(context.Background(), "example.com")

	require.NoError(t, err)
	assert.Len(t, resp.Records, 1)

	record := resp.Records["64087"]
	assert.Equal(t, "64087", record.KeyTag)
	assert.Equal(t, DNSSECAlgorithmECDSAP256SHA256, record.Alg)
	assert.True(t, record.Alg.IsValid())
	assert.Equal(t, DNSSECDigestTypeSHA256, record.DigestType)
	assert.True(t, record.DigestType.IsValid())
	assert.NotNil(t, record.PubKey)
	assert.Contains(t, *record.PubKey, "mdsswUyr3DPW")
}

func TestDns_GetDnssecRecords_Empty(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/dns/getDnssecRecords/example.com", "POST", "/dns/getDnssecRecords/empty.http")

	resp, err := client.DNS.GetDNSSECRecords(context.Background(), "example.com")

	require.NoError(t, err)
	assert.Nil(t, resp.Records)
}

// The five optional key-data fields had no fixture and no assertion, so any of
// their JSON names could have been wrong without the suite noticing.
func TestDNS_CreateDNSSECRecord_KeyDataFields(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	var body map[string]any
	mux.HandleFunc("/dns/createDnssecRecord/example.com", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS"}`)
	})

	_, err := client.DNS.CreateDNSSECRecord(context.Background(), "example.com", &DNSSECRecord{
		KeyTag:          "64087",
		Alg:             DNSSECAlgorithmECDSAP256SHA256,
		DigestType:      DNSSECDigestTypeSHA256,
		Digest:          "ABCD1234",
		MaxSigLife:      Ptr("3600"),
		KeyDataFlags:    Ptr("257"),
		KeyDataProtocol: Ptr("3"),
		KeyDataAlgo:     Ptr(DNSSECAlgorithmECDSAP256SHA256),
		KeyDataPubKey:   Ptr("mdsswUyr3DPW132mOi8V9xESWE8jTo0dxCjjnopKl+GqJxpVXckHAeF+KkxLbxILfDLUT0rAK9iUzy1L53eKGQ=="),
	})

	require.NoError(t, err)
	assert.Equal(t, "3600", body["maxSigLife"])
	assert.Equal(t, "257", body["keyDataFlags"])
	assert.Equal(t, "3", body["keyDataProtocol"])
	assert.Equal(t, "13", body["keyDataAlgo"])
	assert.Contains(t, body["keyDataPubKey"], "mdsswUyr")
}

// The optional fields must not appear when unset, so a minimal DS submission is
// not rejected for carrying empty key data.
func TestDNS_CreateDNSSECRecord_OmitsUnsetOptionalFields(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	var body map[string]any
	mux.HandleFunc("/dns/createDnssecRecord/example.com", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS"}`)
	})

	_, err := client.DNS.CreateDNSSECRecord(context.Background(), "example.com", &DNSSECRecord{
		KeyTag: "64087", Alg: DNSSECAlgorithmECDSAP256SHA256,
		DigestType: DNSSECDigestTypeSHA256, Digest: "ABCD1234",
	})

	require.NoError(t, err)
	for _, key := range []string{"maxSigLife", "keyDataFlags", "keyDataProtocol", "keyDataAlgo", "keyDataPubKey", "pubKey"} {
		assert.NotContains(t, body, key)
	}
}

// pubKey is reported only on retrieval and is not a field of the create body, so a
// record read from GetDNSSECRecords must not carry it into the request.
func TestDnsService_CreateDnssecRecordOmitsPubKey(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	var body map[string]any
	mux.HandleFunc("/dns/createDnssecRecord/example.com", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS"}`)
	})

	var records DNSSECRecords
	require.NoError(t, json.Unmarshal([]byte(
		`{"12345":{"keyTag":"12345","alg":"13","digestType":"2","digest":"ABCD","pubKey":"AwEAAc"}}`), &records))

	record := records["12345"]
	require.NotNil(t, record.PubKey, "the retrieved record does carry pubKey")

	_, err := client.DNS.CreateDNSSECRecord(context.Background(), "example.com", &record)
	require.NoError(t, err)

	assert.NotContains(t, body, "pubKey")
	assert.Equal(t, "12345", body["keyTag"])
	assert.Equal(t, "13", body["alg"])

	// The caller's record is left as it was.
	assert.NotNil(t, record.PubKey)
}

// The three DNSSEC methods had no error-path coverage.
func TestDNS_DNSSECErrorPaths(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	for _, path := range []string{
		"/dns/getDnssecRecords/example.com",
		"/dns/createDnssecRecord/example.com",
		"/dns/deleteDnssecRecord/example.com/64087",
	} {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"status":"ERROR","message":"Domain is not opted in to API access.","code":"DOMAIN_NOT_ALLOWED"}`)
		})
	}

	ctx := context.Background()
	calls := map[string]func() error{
		"GetDNSSECRecords": func() error { _, err := client.DNS.GetDNSSECRecords(ctx, "example.com"); return err },
		"CreateDNSSECRecord": func() error {
			_, err := client.DNS.CreateDNSSECRecord(ctx, "example.com", &DNSSECRecord{
				KeyTag: "64087", Alg: DNSSECAlgorithmECDSAP256SHA256,
				DigestType: DNSSECDigestTypeSHA256, Digest: "AB",
			})
			return err
		},
		"DeleteDNSSECRecord": func() error {
			_, err := client.DNS.DeleteDNSSECRecord(ctx, "example.com", "64087")
			return err
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			var apiErr *ErrorResponse
			require.ErrorAs(t, call(), &apiErr)
			assert.Equal(t, ErrCodeDomainNotAllowed, apiErr.Code)
		})
	}
}
