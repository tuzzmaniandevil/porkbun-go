package porkbun

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
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

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Equal(t, "2404:4400:5401:d900:6b19:e84:33cb:cd66", resp.YourIP)
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

	_, err := client.Ping(nil)

	assert.Error(t, err)
	assert.Equal(t, "context must be non-nil", err.Error())
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
	assert.Equal(t, "invalid character 'S' looking for beginning of value", err.Error())
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
	assert.Equal(t, "unexpected end of JSON input", err.Error())
}

func TestPing_Error_InvalidBody(t *testing.T) {
	setupMockServer(false)
	defer teardownMockServer()

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Something went wrong")
	})

	_, err := client.Ping(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP error 400: Bad Request")
}

func TestPing_EmptyResponse(t *testing.T) {
	setupMockServer(false)
	defer teardownMockServer()

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	_, err := client.Ping(context.Background())

	assert.Error(t, err)
	assert.Equal(t, "unexpected end of JSON input", err.Error())
}

func TestPing_PartialBody(t *testing.T) {
	setupMockServer(false)
	defer teardownMockServer()

	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"SUCCESS`)
	})

	_, err := client.Ping(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
}
