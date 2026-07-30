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

func TestDomainsService_GetNameServers_Success(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/getNs/example.com", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/domains/getNameServers-success.http")
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

	resp, err := client.Domains.GetNameServers(context.Background(), "example.com")

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.NotEmpty(t, resp.NS)

	assert.Contains(t, resp.NS, "fortaleza.ns.porkbun.com")
}

func TestDomainsService_GetNameServers_Error(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/getNs/example.com", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/domains/getNameServers-error.http")
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

	_, err := client.Domains.GetNameServers(context.Background(), "example.com")

	testErrorResponse(t, err)
}

func TestDomainsService_UpdateNameServers_Success(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/updateNs/example.com", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/domains/updateNameServers-success.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)
		testCredentials(t, r)

		expectedBody := map[string]interface{}{
			"apikey":       "1234",
			"secretapikey": "5678",
			"ns":           []interface{}{"s1.example.com", "s2.example.com"},
		}
		testRequestJSON(t, r, expectedBody)

		for k, values := range httpResponse.Header {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}

		w.WriteHeader(httpResponse.StatusCode)
		_, err := io.Copy(w, httpResponse.Body)

		require.NoError(t, err)
	})

	resp, err := client.Domains.UpdateNameServers(context.Background(), "example.com", NameServers{"s1.example.com", "s2.example.com"})

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

func TestDomainsService_UpdateNameServers_Error(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/updateNs/example.com", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/domains/updateNameServers-error.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)
		testCredentials(t, r)

		expectedBody := map[string]interface{}{
			"apikey":       "1234",
			"secretapikey": "5678",
			"ns":           []interface{}{"ns1.example.com"},
		}
		testRequestJSON(t, r, expectedBody)

		for k, values := range httpResponse.Header {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}

		w.WriteHeader(httpResponse.StatusCode)
		_, err := io.Copy(w, httpResponse.Body)

		require.NoError(t, err)
	})

	_, err := client.Domains.UpdateNameServers(context.Background(), "example.com", NameServers{"ns1.example.com"})

	testErrorResponse(t, err)
}

// An empty list is never a valid update, so it is rejected before a request is
// sent rather than costing a round trip to learn the same thing.
func TestDomainsService_UpdateNameServers_EmptyList(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/updateNs/example.com", func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request should be sent for an empty name server list")
	})

	_, err := client.Domains.UpdateNameServers(context.Background(), "example.com", NameServers{})

	assert.Error(t, err)
}

func TestDomainsService_GetNameServers_InvalidResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/getNs/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS","ns":"not-a-list"}`)
	})

	_, err := client.Domains.GetNameServers(context.Background(), "example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "json: cannot unmarshal")
}

func TestDomainsService_GetNameServers_PostFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/getNs/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"status":"ERROR","message":"Internal Server Error"}`)
	})

	_, err := client.Domains.GetNameServers(context.Background(), "example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500: Internal Server Error")
}

func TestDomainsService_UpdateNameServers_PostFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/updateNs/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"status":"ERROR","message":"Internal Server Error"}`)
	})

	_, err := client.Domains.UpdateNameServers(context.Background(), "example.com", NameServers{"ns1.example.com", "ns2.example.com"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500: Internal Server Error")
}

// The v3.6 dry run applies to nameserver updates too, so the response has to
// carry the verdict and not just SUCCESS.
func TestDomainsService_UpdateNameServers_DryRun(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/updateNs/example.com", "POST", "/domains/updateNameServers-dryrun.http",
		func(data map[string]interface{}) {
			assert.Equal(t, true, data["dryRun"])
			assert.Equal(t, []interface{}{"ns1.example.com"}, data["ns"])
		})

	resp, err := client.Domains.UpdateNameServers(context.Background(), "example.com",
		NameServers{"ns1.example.com"}, WithDryRun())

	require.NoError(t, err)
	assert.True(t, resp.DryRun)
	assert.True(t, resp.WouldSucceed)
}
