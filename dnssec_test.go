package porkbun

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDnssecAlgorithm_IsValid(t *testing.T) {
	tests := []struct {
		name string
		da   DnssecAlgorithm
		want bool
	}{
		{"RSA/MD5", DnssecAlgorithmRsaMd5, true},
		{"DSA/SHA-1", DnssecAlgorithmDsaSha1, true},
		{"RSA/SHA-1", DnssecAlgorithmRsaSha1, true},
		{"DSA/SHA-1", DnssecAlgorithmDsaNsec3Sha1, true},
		{"RSA/SHA-256", DnssecAlgorithmRsaSha256, true},
		{"RSA/SHA-512", DnssecAlgorithmRsaSha512, true},
		{"GOST R 34.10-2001", DnssecAlgorithmGostR34111994, true},
		{"ECDSA/SHA-256", DnssecAlgorithmEcdsaSha256, true},
		{"ECDSA/SHA-384", DnssecAlgorithmEcdsaSha384, true},
		{"ED25519", DnssecAlgorithmEd25519, true},
		{"ED448", DnssecAlgorithmEd448, true},
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
		dt   DnssecDigestType
		want bool
	}{
		{"SHA-1", DnssecDigestTypeSha1, true},
		{"SHA-256", DnssecDigestTypeSha256, true},
		{"GOST R 34.10-2001", DnssecDigestTypeGostR34111994, true},
		{"SHA-384", DnssecDigestTypeSha384, true},
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

		assert.NoError(t, err)
	})

	resp, err := client.Dns.GetDnssecRecords(context.Background(), "example.com")

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.NotEmpty(t, resp.Records)

	record := resp.Records["64087"]
	assert.Equal(t, "64087", record.KeyTag)
	assert.Equal(t, DnssecAlgorithmEcdsaSha256, record.Alg)
	assert.Equal(t, DnssecDigestTypeSha256, record.DigestType)
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

		assert.NoError(t, err)
	})

	resp, err := client.Dns.CreateDnssecRecord(context.Background(), "example.com", &DnssecRecordData{
		KeyTag:     "64087",
		Alg:        "13",
		DigestType: "2",
		Digest:     "15E445BD08128BDC213E25F1C8227DF4CB35186CAC701C1C335B2C406D5530DC",
	})

	assert.NoError(t, err)
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

		assert.NoError(t, err)
	})

	resp, err := client.Dns.DeleteDnssecRecord(context.Background(), "example.com", "64087")

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}
