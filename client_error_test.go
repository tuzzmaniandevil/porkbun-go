package porkbun

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An error body carries an error shape, whose fields can clash with the success
// shape the response type expects. The error the API reported has to survive
// that, or a caller loses the code it branches on.
func TestClient_APIErrorInsideOKSurvivesADecodeFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/updateAutoRenew/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// results is typed as a map, and arrives here as a string.
		_, _ = fmt.Fprint(w, `{"status":"ERROR","message":"Domain is not opted in to API access.",`+
			`"code":"DOMAIN_NOT_ALLOWED","results":"none"}`)
	})

	resp, err := client.Domains.UpdateAutoRenew(context.Background(), AutoRenewOn, []string{"example.com"})

	var apiErr *ErrorResponse
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, ErrCodeDomainNotAllowed, apiErr.Code)
	assert.Equal(t, "Domain is not opted in to API access.", apiErr.Message)
	assert.Equal(t, StatusError, resp.Status)
}

// A malformed success body is still a decode error: there is no API error hiding
// behind it to report instead.
func TestClient_DecodeFailureOnSuccessIsReturned(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/getNs/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS","ns":"not-an-array"}`)
	})

	_, err := client.Domains.GetNameServers(context.Background(), "example.com")

	require.Error(t, err)
	var apiErr *ErrorResponse
	assert.False(t, errors.As(err, &apiErr), "a decode failure is not an API error")
}

// The API allocates the record id. Sending the id a record already carries asks
// it to create a record that exists, so a record read back from GetRecords must
// be safe to pass straight to CreateRecord.
func TestDnsService_CreateRecordOmitsTheRecordID(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/create/example.com", func(w http.ResponseWriter, r *http.Request) {
		data, err := getRequestJSON(r)
		require.NoError(t, err)
		assert.NotContains(t, data, "id")
		assert.Equal(t, "www", data["name"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS","id":"123"}`)
	})

	record := &DNSRecord{ID: 999, Name: "www", Type: A, Content: "1.2.3.4", TTL: 600}
	resp, err := client.DNS.CreateRecord(context.Background(), "example.com", record)

	require.NoError(t, err)
	assert.Equal(t, int64(123), resp.ID.Int64())
	assert.Equal(t, FlexInt64(999), record.ID, "the caller's record is left alone")
}

// The response carries the reported failure whichever way it arrived, so Status,
// Code and Message can be read off it for a 4xx as well as for an ERROR inside a
// 200.
func TestClient_ResponseCarriesTheErrorOnANon200(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/get/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"status":"ERROR","message":"Domain not found.","code":"DOMAIN_NOT_FOUND"}`)
	})

	resp, err := client.Domains.GetDomain(context.Background(), "example.com", nil)

	var apiErr *ErrorResponse
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, ErrCodeDomainNotFound, apiErr.Code)

	assert.Equal(t, StatusError, resp.Status)
	assert.Equal(t, ErrCodeDomainNotFound, resp.Code)
	assert.Equal(t, "Domain not found.", resp.Message)
	require.NotNil(t, resp.HTTPResponse)
	assert.Equal(t, http.StatusNotFound, resp.HTTPResponse.StatusCode)
}

// A gateway in front of the API answers with HTML for an unrecognised path. That
// still has to come back as an *ErrorResponse, or the status code and errors.As
// go out of reach.
func TestClient_NonJSONErrorBodyStillYieldsAnErrorResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/get/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = fmt.Fprint(w, `<html><body>502 Bad Gateway</body></html>`)
	})

	_, err := client.Domains.GetDomain(context.Background(), "example.com", nil)

	var apiErr *ErrorResponse
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, StatusError, apiErr.Status)
	assert.Equal(t, http.StatusText(http.StatusBadGateway), apiErr.Message)
}

// An error body that is valid JSON of the wrong shape half-decodes: encoding/json
// assigns the fields it reaches before failing. Those cannot be trusted, so they
// must be discarded rather than reported alongside the status text.
//
// A syntax error would not exercise this: Unmarshal validates the whole document
// before assigning anything, so nothing is ever populated on that path.
func TestClient_HalfDecodedErrorFieldsAreDiscarded(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/get/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		// status, message and code decode; next_action then fails on its type.
		_, _ = fmt.Fprint(w, `{"status":"ERROR","message":"half decoded","code":"INVALID_DOMAIN","next_action":"not an object"}`)
	})

	_, err := client.Domains.GetDomain(context.Background(), "example.com", nil)

	var apiErr *ErrorResponse
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusText(http.StatusBadGateway), apiErr.Message, "the untrusted message must be replaced")
	assert.Empty(t, apiErr.Code, "a code from a body that failed to decode cannot be trusted")
	assert.Nil(t, apiErr.NextAction, "a partially built NextAction must not reach the caller")
}

// An empty error body leaves the status text as the message, rather than counting
// as a body that failed to decode.
func TestClient_EmptyErrorBodyDescribesTheStatus(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/get/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := client.Domains.GetDomain(context.Background(), "example.com", nil)

	var apiErr *ErrorResponse
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusText(http.StatusTooManyRequests), apiErr.Message)
}
