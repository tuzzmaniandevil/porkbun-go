package porkbun

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPorkbun_NewClient(t *testing.T) {
	var httpClient HTTPClient
	httpClient = http.DefaultClient
	client := NewClient(&Options{
		HttpClient:   &httpClient,
		ApiKey:       "1234",
		SecretApiKey: "5678",
		UserAgent:    "CustomAgent/1",
		IPv4Only:     true,
	})

	assert.Equal(t, ipv4OnlyBaseURL, client.baseURL)
	assert.Equal(t, "1234", client.apiKey)
	assert.Equal(t, "5678", client.secret)
	assert.Equal(t, "CustomAgent/1", client.userAgent)
}

func TestPorkbun_NewRequest(t *testing.T) {
	client := NewClient(&Options{})

	req, _ := client.newRequest("POST", "/somepath", nil)

	assert.Equal(t, defaultBaseURL+"/somepath", req.URL.String())
}

func TestPorkbun_NewRequest_UserAgent(t *testing.T) {
	client := NewClient(&Options{
		UserAgent: "UserAgent/23",
	})

	req, _ := client.newRequest("POST", "/somepath", nil)

	assert.Equal(t, defaultBaseURL+"/somepath", req.URL.String())
	assert.Equal(t, "UserAgent/23 "+defaultUserAgent, req.Header.Get("User-Agent"))
}

func TestPorkbun_NewRequest_InvalidMethod(t *testing.T) {
	client := NewClient(&Options{})

	_, err := client.newRequest("💩", "/", nil)

	assert.Error(t, err)
}

func TestPorkbun_MakeRequest_InvalidMethod(t *testing.T) {
	client := NewClient(&Options{})

	_, err := client.makeRequest(context.Background(), "💩", "/", nil, nil)

	assert.Error(t, err)
}

type InvalidObject struct{}

func (o *InvalidObject) MarshalJSON() ([]byte, error) {
	return nil, errors.New("Invalid Object")
}

func TestPorkbun_MakeRequest_InvalidPayload(t *testing.T) {
	client := NewClient(&Options{})

	_, err := client.makeRequest(context.Background(), "POST", "/", &InvalidObject{}, nil)

	assert.Error(t, err)
}

func TestPorkbun_MakeRequest_404NotFound(t *testing.T) {
	setupMockServer(false)
	defer teardownMockServer()

	mux.HandleFunc("/test404", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := client.makeRequest(context.Background(), "GET", "/test404", nil, nil)
	assert.Error(t, err)
	assert.IsType(t, &ErrorResponse{}, err)
}

func TestPorkbun_MakeRequest_500InternalServerError(t *testing.T) {
	setupMockServer(false)
	defer teardownMockServer()

	mux.HandleFunc("/test500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := client.makeRequest(context.Background(), "GET", "/test500", nil, nil)
	assert.Error(t, err)
	assert.IsType(t, &ErrorResponse{}, err)
}

func TestPorkbun_MakeRequest_NilObject(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/somepath", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"SUCCESS"}`)
	})

	resp, err := client.makeRequest(context.Background(), "POST", "/somepath", nil, nil)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPorkbun_Request_ReadBodyError(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/somepath", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush() // Simulate incomplete response
		server.CloseClientConnections()
	})

	req, err := client.newRequest("POST", "/somepath", nil)
	assert.NoError(t, err)

	var obj map[string]interface{}
	_, err = client.request(context.Background(), req, &obj)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected EOF")
}
