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

func TestSslService_RetrieveSuccess(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/ssl/retrieve/example.com", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/ssl/retrieve-success.http")
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
		_, _ = io.Copy(w, httpResponse.Body)
	})

	resp, err := client.SSL.Retrieve(context.Background(), "example.com")

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)

	// The endpoint's entire payload. Without these, all three JSON tags could be
	// renamed with the suite green.
	assert.Contains(t, resp.CertificateChain, "-----BEGIN CERTIFICATE-----")
	assert.Contains(t, resp.PrivateKey, "-----BEGIN PRIVATE KEY-----")
	assert.Contains(t, resp.PublicKey, "-----BEGIN PUBLIC KEY-----")

	respHeaders := resp.HTTPResponse.Header
	assert.Equal(t, "openresty", respHeaders.Get("server"))
}

func TestSslService_RetrieveError(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/ssl/retrieve/example.com", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/ssl/retrieve-error.http")
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
		_, _ = io.Copy(w, httpResponse.Body)
	})

	_, err := client.SSL.Retrieve(context.Background(), "example.com")

	testErrorResponse(t, err)
}

func TestSslService_RetrieveEmptyResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/ssl/retrieve/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Simulate an empty response body
	})

	_, err := client.SSL.Retrieve(context.Background(), "example.com")

	assert.Error(t, err)
	assert.ErrorContains(t, err, "decoding response body: unexpected end of JSON input")
}

func TestSslService_RetrieveInvalidResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/ssl/retrieve/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Simulate invalid/malformed JSON response
		_, _ = fmt.Fprint(w, "Invalid JSON")
	})

	_, err := client.SSL.Retrieve(context.Background(), "example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
}

func TestSslService_RetrieveNetworkError(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	// Simulate a network error by closing the server immediately
	server.Close()

	_, err := client.SSL.Retrieve(context.Background(), "example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestSslService_Retrieve_PostFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/ssl/retrieve/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"status":"ERROR","message":"Internal Server Error"}`)
	})

	_, err := client.SSL.Retrieve(context.Background(), "example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500: Internal Server Error")
}

func TestSsl_Retrieve_NotReady(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/ssl/retrieve/example.com", "POST", "/ssl/retrieve-notready.http")

	_, err := client.SSL.Retrieve(context.Background(), "example.com")

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrorCode("THE_SSL_CERTIFICATE_IS_NOT_READY_FOR_THIS_DOMAIN"), errResponse.Code)
}
