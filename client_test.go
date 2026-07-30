package porkbun

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPorkbun_NewClient(t *testing.T) {
	client := NewClient(&Options{
		HTTPClient:   http.DefaultClient,
		APIKey:       "1234",
		SecretAPIKey: "5678",
		UserAgent:    "CustomAgent/1",
		IPv4Only:     true,
	})

	assert.Equal(t, ipv4OnlyBaseURL, client.baseURL)
	assert.Equal(t, "1234", client.apiKey)
	assert.Equal(t, "5678", client.secret)
	assert.Equal(t, "CustomAgent/1", client.userAgent)
}

// BaseURL points the client at a mock or a proxy, and otherwise selects the
// Porkbun endpoint according to IPv4Only.
func TestPorkbun_NewClient_BaseURL(t *testing.T) {
	tests := []struct {
		name    string
		options Options
		want    string
	}{
		{"default", Options{}, defaultBaseURL},
		{"ipv4 only", Options{IPv4Only: true}, ipv4OnlyBaseURL},
		{"override", Options{BaseURL: "http://127.0.0.1:8080/v3"}, "http://127.0.0.1:8080/v3"},
		{"trailing slash trimmed", Options{BaseURL: "http://127.0.0.1:8080/v3/"}, "http://127.0.0.1:8080/v3"},
		{"override wins over ipv4 only", Options{BaseURL: "http://mock", IPv4Only: true}, "http://mock"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewClient(&tt.options).baseURL)
		})
	}
}

// The zero http.Client waits forever, so the one NewClient builds carries a
// timeout. A supplied client is left exactly as it came.
func TestPorkbun_NewClient_Timeout(t *testing.T) {
	built, ok := NewClient(nil).httpClient.(*http.Client)
	require.True(t, ok)
	assert.Equal(t, DefaultTimeout, built.Timeout)

	supplied := &http.Client{}
	assert.Same(t, supplied, NewClient(&Options{HTTPClient: supplied}).httpClient)
	assert.Zero(t, supplied.Timeout)
}

func TestPorkbun_NewRequest(t *testing.T) {
	client := NewClient(&Options{})

	req, _ := client.newRequest("POST", "/somepath", nil, newRequestConfig(nil))

	assert.Equal(t, defaultBaseURL+"/somepath", req.URL.String())
}

func TestPorkbun_NewRequest_UserAgent(t *testing.T) {
	client := NewClient(&Options{
		UserAgent: "UserAgent/23",
	})

	req, _ := client.newRequest("POST", "/somepath", nil, newRequestConfig(nil))

	assert.Equal(t, defaultBaseURL+"/somepath", req.URL.String())
	assert.Equal(t, "UserAgent/23 "+defaultUserAgent, req.Header.Get("User-Agent"))
}

func TestPorkbun_NewRequest_InvalidMethod(t *testing.T) {
	client := NewClient(&Options{})

	// A space is not valid in an HTTP method token.
	_, err := client.newRequest("BAD METHOD", "/", nil, newRequestConfig(nil))

	assert.Error(t, err)
}

func TestPorkbun_MakeRequest_InvalidMethod(t *testing.T) {
	client := NewClient(&Options{})

	_, err := client.makeRequest(context.Background(), "BAD METHOD", "/", nil, nil)

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
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS"}`)
	})

	resp, err := client.makeRequest(context.Background(), "POST", "/somepath", nil, nil)

	require.NoError(t, err)
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

	req, err := client.newRequest("POST", "/somepath", nil, newRequestConfig(nil))
	require.NoError(t, err)

	var obj map[string]interface{}
	_, err = client.request(context.Background(), req, &obj)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected EOF")
}

func TestPorkbun_HeaderAuth(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	req, err := client.newRequest("GET", "/account/balance", nil, newRequestConfig(nil))

	require.NoError(t, err)
	assert.Equal(t, "1234", req.Header.Get("X-API-Key"))
	assert.Equal(t, "5678", req.Header.Get("X-Secret-API-Key"))
}

func TestPorkbun_HeaderAuth_SkippedForApiKeyFlow(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	req, err := client.newRequest("POST", "/apikey/request", nil, newRequestConfig([]RequestOption{withoutHeaderAuth()}))

	require.NoError(t, err)
	assert.Empty(t, req.Header.Get("X-API-Key"))
	assert.Empty(t, req.Header.Get("X-Secret-API-Key"))
}

// Header auth needs both halves, so a client holding only a public key sends
// neither header rather than a key with an empty secret.
func TestPorkbun_HeaderAuth_SkippedWithoutSecret(t *testing.T) {
	setupMockServer(false)
	defer teardownMockServer()

	client.apiKey = "1234"

	req, err := client.newRequest("GET", "/account/balance", nil, newRequestConfig(nil))

	require.NoError(t, err)
	assert.Empty(t, req.Header.Get("X-API-Key"))
	assert.Empty(t, req.Header.Get("X-Secret-API-Key"))
}

// The API applies header auth only when the body carries no credentials, so a
// POST whose payload already carries them must not send the secret twice.
func TestPorkbun_HeaderAuth_SkippedWhenBodyCarriesCredentials(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	req, err := client.newRequest("POST", "/ssl/retrieve/example.com", &sslRetrieveRequest{}, newRequestConfig(nil))

	require.NoError(t, err)
	assert.Empty(t, req.Header.Get("X-API-Key"), "the body already carries the credentials")
	assert.Empty(t, req.Header.Get("X-Secret-API-Key"))

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"secretapikey":"5678","apikey":"1234"}`, string(body))
}

// Options.HTTPClient exists to be substituted, and a double that answers with the
// shape most hand-written mocks use must not panic inside the SDK.
func TestPorkbun_MalformedHTTPClientResponsesAreErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("nil body", func(t *testing.T) {
		c := NewClient(&Options{HTTPClient: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Request: r}, nil
		})})

		resp, err := c.Ping(ctx)

		// An empty body is a decode failure, not a panic.
		require.Error(t, err)
		assert.ErrorContains(t, err, "decoding response body")
		assert.NotNil(t, resp)
	})

	t.Run("nil response and nil error", func(t *testing.T) {
		c := NewClient(&Options{HTTPClient: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, nil
		})})

		_, err := c.Ping(ctx)

		assert.ErrorContains(t, err, "nil response and a nil error")
	})
}

// HTTPResponse is handed to the caller after the body has been drained, so the
// Body left on it must read as empty rather than as a closed reader.
func TestPorkbun_HTTPResponseBodyIsReadableAndEmpty(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/ping", "POST", "/ping/success.http")

	resp, err := client.Ping(context.Background())
	require.NoError(t, err)
	require.NotNil(t, resp.HTTPResponse)

	body, err := io.ReadAll(resp.HTTPResponse.Body)
	require.NoError(t, err, "reading the drained body must not fail")
	assert.Empty(t, body)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func TestPorkbun_IdempotencyKey(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	cfg := newRequestConfig([]RequestOption{WithIdempotencyKey("a1b2c3d4")})
	req, err := client.newRequest("POST", "/domain/create/example.com", nil, cfg)

	require.NoError(t, err)
	assert.Equal(t, "a1b2c3d4", req.Header.Get("Idempotency-Key"))
}

func TestPorkbun_IdempotencyKey_NotSetByDefault(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	req, err := client.newRequest("POST", "/domain/create/example.com", nil, newRequestConfig(nil))

	require.NoError(t, err)
	assert.Empty(t, req.Header.Get("Idempotency-Key"))
}

// WithDryRun applies to exactly the call it is passed to, so a request built
// without it must not be flagged.
func TestPorkbun_WithDryRun(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	flagged := &createDomainRequest{}
	_, err := client.newRequest("POST", "/domain/create/example.com", flagged,
		newRequestConfig([]RequestOption{WithDryRun()}))
	require.NoError(t, err)
	assert.True(t, flagged.DryRun)

	plain := &createDomainRequest{}
	_, err = client.newRequest("POST", "/domain/create/example.com", plain, newRequestConfig(nil))
	require.NoError(t, err)
	assert.False(t, plain.DryRun)
}

// An error response whose body is not JSON falls back to the HTTP status.
func TestPorkbun_CheckResponse_UndecodableErrorBody(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/listAll", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/error-invalidbody.http")
		assert.NotNil(t, httpResponse)

		w.WriteHeader(httpResponse.StatusCode)
		_, _ = io.Copy(w, httpResponse.Body)
	})

	_, err := client.Domains.ListDomains(context.Background(), nil)

	// An HTML or plain-text body from a gateway must not downgrade the error to
	// an untyped one: errors.As and the status code stay usable.
	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, http.StatusBadRequest, errResponse.HTTPResponse.StatusCode)
	assert.Equal(t, "Bad Request", errResponse.Message)
	assert.Empty(t, errResponse.Code, "a body that did not decode must not yield a code")
}
