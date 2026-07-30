package porkbun

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The methods below take a pointer or slice that the request cannot be built
// without, so a nil from the caller must return an error rather than panicking
// inside the SDK or posting a bodyless request the API rejects.
//
// The response comes back non-nil alongside the error, as it does on every other
// failure path, so a caller that inspects it after checking err cannot nil-deref.
func TestMissingRequiredArgumentsReturnErrors(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no request should be sent: %s %s", r.Method, r.URL.Path)
	})

	t.Run("CreateDomain", func(t *testing.T) {
		resp, err := client.Domains.CreateDomain(context.Background(), "example.com", nil)

		require.Error(t, err)
		assert.NotNil(t, resp, "the response must be usable alongside the error")
	})

	t.Run("UpdateNameServers", func(t *testing.T) {
		resp, err := client.Domains.UpdateNameServers(context.Background(), "example.com", nil)

		require.Error(t, err)
		assert.NotNil(t, resp, "the response must be usable alongside the error")
	})

	// ips is required by the API, but the field is omitempty so that the same
	// request type can serve DeleteGlueRecord. An empty slice would therefore
	// send no ips at all and be rejected by the API rather than here.
	t.Run("CreateGlueRecord", func(t *testing.T) {
		resp, err := client.Domains.CreateGlueRecord(context.Background(), "example.com", "ns1", nil)

		require.Error(t, err)
		assert.NotNil(t, resp, "the response must be usable alongside the error")
	})

	t.Run("UpdateGlueRecord", func(t *testing.T) {
		resp, err := client.Domains.UpdateGlueRecord(context.Background(), "example.com", "ns1", []string{})

		require.Error(t, err)
		assert.NotNil(t, resp, "the response must be usable alongside the error")
	})

	// A nil embedded pointer is dropped silently by encoding/json, so without the
	// guard these would post credentials with no payload and fail at the API.
	t.Run("CreateRecord", func(t *testing.T) {
		resp, err := client.DNS.CreateRecord(context.Background(), "example.com", nil)

		require.Error(t, err)
		assert.NotNil(t, resp, "the response must be usable alongside the error")
	})

	t.Run("EditRecord", func(t *testing.T) {
		resp, err := client.DNS.EditRecord(context.Background(), "example.com", 1234, nil)

		require.Error(t, err)
		assert.NotNil(t, resp, "the response must be usable alongside the error")
	})

	t.Run("EditRecordByType", func(t *testing.T) {
		resp, err := client.DNS.EditRecordByType(context.Background(), "example.com", A, nil, nil)

		require.Error(t, err)
		assert.NotNil(t, resp, "the response must be usable alongside the error")
	})

	t.Run("CreateDNSSECRecord", func(t *testing.T) {
		resp, err := client.DNS.CreateDNSSECRecord(context.Background(), "example.com", nil)

		require.Error(t, err)
		assert.NotNil(t, resp, "the response must be usable alongside the error")
	})

	t.Run("AddURLForward", func(t *testing.T) {
		resp, err := client.Domains.AddURLForward(context.Background(), "example.com", nil)

		require.Error(t, err)
		assert.NotNil(t, resp, "the response must be usable alongside the error")
	})
}

// NewClient(nil) is the natural way to ask for a client with no credentials,
// which is all Ping, IP and ListPricing need.
func TestNewClientWithoutOptions(t *testing.T) {
	c := NewClient(nil)

	require.NotNil(t, c)
	assert.Equal(t, defaultBaseURL, c.baseURL)
	assert.Empty(t, c.apiKey)
	assert.NotNil(t, c.Domains)
	assert.NotNil(t, c.Webhooks)
}

// Path segments come from caller input, so a domain carrying a slash or a
// traversal must not be able to retarget the request at another endpoint.
func TestPathSegmentsAreEscaped(t *testing.T) {
	assert.Equal(t, "/dns/retrieve/..%2F..%2Faccount%2Fbalance",
		apiPath("dns", "retrieve", "../../account/balance"))
	assert.Equal(t, "/dns/retrieve/exa%20mple.com",
		apiPath("dns", "retrieve", "exa mple.com"))

	// Ordinary values are unchanged, including the dots in a hostname, the
	// underscores in an SRV name, and the "*" of a wildcard record.
	assert.Equal(t, "/dns/edit/example.com/123", apiPath("dns", "edit", "example.com", int64(123)))
	assert.Equal(t, "/dns/retrieveByNameType/example.com/SRV/_sip._tcp",
		apiPath("dns", "retrieveByNameType", "example.com", SRV, "_sip._tcp"))
	assert.Equal(t, "/dns/retrieveByNameType/example.com/A/*",
		apiPath("dns", "retrieveByNameType", "example.com", A, String("*")),
		"escaping the wildcard name would break wildcard record lookups")

	// A nil optional segment is omitted rather than rendered.
	assert.Equal(t, "/dns/retrieve/example.com",
		apiPath("dns", "retrieve", "example.com", (*int64)(nil)))
}

// recordingClient captures the request line without sending it, which is the
// only place the escaping can be judged: an httptest ServeMux would redirect on
// the decoded path and hide the result.
type recordingClient struct{ requestURI string }

func (c *recordingClient) Do(req *http.Request) (*http.Response, error) {
	c.requestURI = req.URL.RequestURI()

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
		Header:     http.Header{},
		Request:    req,
	}, nil
}

func TestPathTraversalDoesNotRetargetTheRequest(t *testing.T) {
	recorder := &recordingClient{}

	c := NewClient(&Options{HTTPClient: recorder, APIKey: "1234", SecretAPIKey: "5678"})

	_, _ = c.Domains.GetNameServers(context.Background(), "../../account/balance")

	assert.Equal(t, "/api/json/v3/domain/getNs/..%2F..%2Faccount%2Fbalance", pathOf(t, recorder.requestURI))
	assert.NotContains(t, recorder.requestURI, "/account/balance",
		"a crafted domain must not be able to retarget the request at another endpoint")
}

// pathOf returns just the path portion of a request URI.
func pathOf(t *testing.T, requestURI string) string {
	t.Helper()

	parsed, err := url.Parse(requestURI)
	require.NoError(t, err)

	return parsed.EscapedPath()
}

// A failed call still carries its HTTP response, so the rate limit headers that
// come with a 429 are reachable from the response and not only from the error.
func TestFailedCallStillAttachesTheHTTPResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/checkDomain/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-API-Version", "3.9")
		w.Header().Set("X-RateLimit-Limit", "20")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "7")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"status":"ERROR","code":"RATE_LIMIT_EXCEEDED","message":"Slow down"}`))
	})

	resp, err := client.Domains.CheckDomain(context.Background(), "example.com")

	testErrorResponse(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse, "the response should carry the HTTP response even on failure")

	assert.Equal(t, http.StatusTooManyRequests, resp.HTTPResponse.StatusCode)
	assert.Equal(t, "3.9", resp.APIVersion())

	limits, ok := resp.RateLimit()
	require.True(t, ok)
	assert.Equal(t, int64(0), limits.Remaining)
	assert.Equal(t, int64(7), limits.Reset)
}

// The specification lets a number of endpoints report a failure as status ERROR
// inside a 200, the DNS and glue writes among them. Reporting that as success
// would tell a caller a record was deleted when it was not.
func TestErrorStatusInsideA200IsAnError(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/delete/example.com/123", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ERROR","code":"RECORD_NOT_FOUND","message":"No such record."}`))
	})

	_, err := client.DNS.DeleteRecord(context.Background(), "example.com", 123)

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrorCode("RECORD_NOT_FOUND"), errResponse.Code)
	assert.Equal(t, "No such record.", errResponse.Message)
	assert.Equal(t, http.StatusOK, errResponse.HTTPResponse.StatusCode)
}

// A status the API uses for something other than failure must still succeed.
// /apikey/retrieve reports PENDING until the user authorizes the request, and
// UpdateAutoRenew reports per-domain failures under a top-level SUCCESS.
func TestNonErrorStatusesAreNotErrors(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureNoAuth(t, "/apikey/retrieve", "POST", "/apikey/retrieve-pending.http", nil)
	handleFixture(t, "/domain/updateAutoRenew/example.com", "POST", "/domains/updateAutoRenew-partial.http")

	pending, err := client.APIKey.Retrieve(context.Background(), "abc", "")
	require.NoError(t, err, "PENDING is not a failure")
	assert.Equal(t, "PENDING", pending.Status)

	partial, err := client.Domains.UpdateAutoRenew(context.Background(), AutoRenewOn, []string{"example.com", "notmine.com"})
	require.NoError(t, err, "a per-domain ERROR under a top-level SUCCESS is not a failed call")
	assert.Equal(t, "ERROR", partial.Results["notmine.com"].Status)
}

// Escaping cannot neutralise a "." or ".." segment, since url.PathEscape leaves a
// dot alone: a domain of ".." would make DeleteRecord ask for "/dns/delete/../5",
// which anything resolving the path before routing it reads as "/dns/5".
func TestClient_RejectsARelativePathSegment(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	var reached string
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reached = r.URL.Path
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS"}`)
	})

	ctx := context.Background()

	for _, domain := range []string{"..", "."} {
		t.Run(domain, func(t *testing.T) {
			_, err := client.DNS.DeleteRecord(ctx, domain, 5)
			assert.ErrorContains(t, err, "would retarget the request")

			_, err = client.DNS.GetRecords(ctx, domain, nil)
			assert.ErrorContains(t, err, "would retarget the request")

			_, err = client.Domains.GetDomain(ctx, domain, nil)
			assert.ErrorContains(t, err, "would retarget the request")
		})
	}

	// A dot inside a segment is a domain name, not a relative reference.
	_, err := client.DNS.GetRecords(ctx, "example.com", nil)
	require.NoError(t, err)
	assert.Equal(t, "/dns/retrieve/example.com", reached)

	// Nor is a subdomain that merely starts with one.
	_, err = client.DNS.GetRecordsByType(ctx, "example.com", TXT, Ptr("_acme-challenge"))
	require.NoError(t, err)
	assert.Equal(t, "/dns/retrieveByNameType/example.com/TXT/_acme-challenge", reached)
}
