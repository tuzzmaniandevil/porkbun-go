package porkbun

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPing_NoAuth(t *testing.T) {
	setupMockServer(false)
	defer teardownMockServer()

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/ping/noauth.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)

		w.WriteHeader(httpResponse.StatusCode)
		_, _ = io.Copy(w, httpResponse.Body)
	})

	_, err := client.Ping(context.Background())

	testErrorResponse(t, err)
}

func TestPing_Success(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/ping/success.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)
		testCredentials(t, r)

		w.WriteHeader(httpResponse.StatusCode)
		_, _ = io.Copy(w, httpResponse.Body)
	})

	resp, err := client.Ping(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Equal(t, "2404:4400:5401:d900:6b19:e84:33cb:cd66", resp.YourIP)
	assert.True(t, resp.CredentialsValid)
}

func TestPing_NoContext(t *testing.T) {
	setupMockServer(false)
	defer teardownMockServer()

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/ping/noauth.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)

		w.WriteHeader(httpResponse.StatusCode)
		_, _ = io.Copy(w, httpResponse.Body)
	})

	// A nil context is exactly what this test asserts against.
	//lint:ignore SA1012 deliberate
	_, err := client.Ping(nil) //nolint:staticcheck

	assert.Error(t, err)
	assert.Equal(t, "porkbun: nil context", err.Error())
}

func TestPing_InvalidBody(t *testing.T) {
	setupMockServer(false)
	defer teardownMockServer()

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/success-invalidbody.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)

		w.WriteHeader(httpResponse.StatusCode)
		_, _ = io.Copy(w, httpResponse.Body)
	})

	_, err := client.Ping(context.Background())

	assert.Error(t, err)
	assert.ErrorContains(t, err, "decoding response body: invalid character 'S' looking for beginning of value")
}

func TestPing_NoBody(t *testing.T) {
	setupMockServer(false)
	defer teardownMockServer()

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/success-nobody.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)

		w.WriteHeader(httpResponse.StatusCode)
		_, _ = io.Copy(w, httpResponse.Body)
	})

	_, err := client.Ping(context.Background())

	assert.Error(t, err)
	assert.ErrorContains(t, err, "decoding response body: unexpected end of JSON input")
}

func TestPing_Error_InvalidBody(t *testing.T) {
	setupMockServer(false)
	defer teardownMockServer()

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, "Something went wrong")
	})

	_, err := client.Ping(context.Background())

	// A non-JSON error body still has to arrive as an *ErrorResponse, so callers
	// can reach the status code without string matching.
	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, http.StatusBadRequest, errResponse.HTTPResponse.StatusCode)
	assert.Equal(t, "Bad Request", errResponse.Message)
}

func TestPing_EmptyResponse(t *testing.T) {
	setupMockServer(false)
	defer teardownMockServer()

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	_, err := client.Ping(context.Background())

	assert.Error(t, err)
	assert.ErrorContains(t, err, "decoding response body: unexpected end of JSON input")
}

func TestPing_PartialBody(t *testing.T) {
	setupMockServer(false)
	defer teardownMockServer()

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS`)
	})

	_, err := client.Ping(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
}

func TestIP(t *testing.T) {
	// /ip needs no credentials at all.
	setupMockServer(false)
	defer teardownMockServer()

	handleFixtureNoAuth(t, "/ip", "GET", "/ip/success.http", nil)

	resp, err := client.IP(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Equal(t, "2406:5a00:8a14:ea00:e0f3:f116:a523:7666", resp.YourIP)
	assert.Equal(t, "2406:5a00:8a14:ea00:e0f3:f116:a523:7666", resp.XForwardedFor)
}

// /ip takes no credentials, so a configured client must not send them: an API
// key restricted by source IP would be rejected from the address being looked up.
func TestIP_SendsNoCredentialsWhenConfigured(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureNoAuth(t, "/ip", "GET", "/ip/success.http", nil)

	resp, err := client.IP(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

// A client with no keys sends an empty body rather than empty credentials, which
// is what selects the API's IP-only response.
func TestPing_NoCredentialsConfigured(t *testing.T) {
	setupMockServer(false)
	defer teardownMockServer()

	handleFixtureNoAuth(t, "/ping", "POST", "/ping/nocredentials.http",
		func(data map[string]interface{}) {
			assert.Empty(t, data)
		})

	resp, err := client.Ping(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.NotEmpty(t, resp.YourIP)
	assert.False(t, resp.CredentialsValid)
}
