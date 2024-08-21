package porkbun

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
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

	resp, err := client.Ssl.Retrieve(context.Background(), "example.com")

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)

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

	_, err := client.Ssl.Retrieve(context.Background(), "example.com")

	testErrorResponse(t, err)
}

func TestSslService_RetrieveEmptyResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/ssl/retrieve/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Simulate an empty response body
	})

	_, err := client.Ssl.Retrieve(context.Background(), "example.com")

	assert.Error(t, err)
	assert.Equal(t, "unexpected end of JSON input", err.Error())
}

func TestSslService_RetrieveInvalidResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/ssl/retrieve/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Simulate invalid/malformed JSON response
		fmt.Fprint(w, "Invalid JSON")
	})

	_, err := client.Ssl.Retrieve(context.Background(), "example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character")
}

func TestSslService_RetrieveNetworkError(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	// Simulate a network error by closing the server immediately
	server.Close()

	_, err := client.Ssl.Retrieve(context.Background(), "example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestSslService_Retrieve_PostFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/ssl/retrieve/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"status":"ERROR","message":"Internal Server Error"}`)
	})

	_, err := client.Ssl.Retrieve(context.Background(), "example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500 Internal Server Error")
}
