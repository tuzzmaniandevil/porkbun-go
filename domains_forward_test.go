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

func TestDomainsService_GetDomainURLForwarding_Success(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/getUrlForwarding/example.com", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/domains/getDomainURLForwarding-success.http")
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

	resp, err := client.Domains.ListURLForwards(context.Background(), "example.com")

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.NotEmpty(t, resp.Forwards)

	forward := resp.Forwards[0]
	assert.Equal(t, "22049216", forward.ID)
	assert.Equal(t, Temporary, forward.Type)
}

func TestDomainsService_AddDomainUrlForward_Success(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/addUrlForward/example.com", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/domains/addUrlForward-success.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)
		testCredentials(t, r)

		expectedBody := map[string]interface{}{
			"apikey":       "1234",
			"secretapikey": "5678",
			"subdomain":    "",
			"location":     "https://porkbun.com",
			"type":         "temporary",
			"includePath":  "yes",
			"wildcard":     "yes",
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

	resp, err := client.Domains.AddURLForward(context.Background(), "example.com", &URLForward{
		Location:    "https://porkbun.com",
		Type:        Temporary,
		IncludePath: "yes",
		Wildcard:    "yes",
	})

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

// The API accepts only alphanumerics and hyphens in subdomain, so a fully
// qualified name carried over from ListURLForwards loses the domain the
// same way a DNS record name does.
func TestDomainsService_AddDomainUrlForward_UnqualifiesSubdomain(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/addUrlForward/example.com", "POST",
		"/domains/addUrlForward-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "www", data["subdomain"])
		})

	forward := &URLForward{Subdomain: "www.example.com", Location: "https://porkbun.com", Type: Temporary}
	_, err := client.Domains.AddURLForward(context.Background(), "example.com", forward)

	require.NoError(t, err)
	// The caller's struct is left alone.
	assert.Equal(t, "www.example.com", forward.Subdomain)
}

func TestDomainsService_AddDomainUrlForward_Error(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/addUrlForward/unknowndomain.com", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/domains/addUrlForward-error.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)
		testCredentials(t, r)

		expectedBody := map[string]interface{}{
			"apikey":       "1234",
			"secretapikey": "5678",
			"subdomain":    "",
			"location":     "https://porkbun.com",
			"type":         "temporary",
			"includePath":  "yes",
			"wildcard":     "yes",
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

	_, err := client.Domains.AddURLForward(context.Background(), "unknowndomain.com", &URLForward{
		Location:    "https://porkbun.com",
		Type:        Temporary,
		IncludePath: "yes",
		Wildcard:    "yes",
	})

	testErrorResponse(t, err)
}

func TestDomainsService_DeleteDomainUrlForward_Success(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/deleteUrlForward/example.com/12345", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/domains/addUrlForward-success.http")
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

	resp, err := client.Domains.DeleteURLForward(context.Background(), "example.com", "12345")

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

func TestDomainsService_GetDomainURLForwarding_InvalidResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/getUrlForwarding/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS","forwards":"invalid"}`)
	})

	_, err := client.Domains.ListURLForwards(context.Background(), "example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "json: cannot unmarshal")
}

func TestDomainsService_GetDomainURLForwarding_EmptyResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/getUrlForwarding/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS","forwards":[]}`)
	})

	resp, err := client.Domains.ListURLForwards(context.Background(), "example.com")

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Empty(t, resp.Forwards)
}

func TestDomainsService_GetDomainURLForwarding_PartialResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/getUrlForwarding/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
            "status":"SUCCESS",
            "forwards":[{
                "id":"22049216",
                "type":"temporary",
                "location":"https://porkbun.com"
            }]
        }`)
	})

	resp, err := client.Domains.ListURLForwards(context.Background(), "example.com")

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Len(t, resp.Forwards, 1)
	assert.Equal(t, "22049216", resp.Forwards[0].ID)
	assert.Equal(t, Temporary, resp.Forwards[0].Type)
	assert.Equal(t, "https://porkbun.com", resp.Forwards[0].Location)
}

func TestDomainsService_DeleteDomainUrlForward_PostFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/deleteUrlForward/example.com/12345", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"status":"ERROR","message":"Internal Server Error"}`)
	})

	_, err := client.Domains.DeleteURLForward(context.Background(), "example.com", "12345")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500: Internal Server Error")
}

// RedirectType carries the exact stored code, so a 307 stays distinguishable
// from a 302 even though both report Type "temporary".
func TestDomains_GetDomainURLForwarding_RedirectType(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/getUrlForwarding/example.com", "POST", "/domains/getDomainURLForwarding-redirecttype.http")

	resp, err := client.Domains.ListURLForwards(context.Background(), "example.com")

	require.NoError(t, err)
	assert.Len(t, resp.Forwards, 2)

	assert.Equal(t, "52511", resp.Forwards[0].ID)
	assert.Equal(t, Temporary, resp.Forwards[0].Type)
	assert.Equal(t, Redirect307, resp.Forwards[0].RedirectType)
	assert.Equal(t, Yes, resp.Forwards[0].Wildcard)

	assert.Equal(t, "shop", resp.Forwards[1].Subdomain)
	assert.Equal(t, Masked, resp.Forwards[1].Type)
	assert.Equal(t, RedirectMasked, resp.Forwards[1].RedirectType)
}

func TestDomains_AddDomainUrlForward_RedirectType(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/addUrlForward/example.com", "POST", "/domains/addUrlForward-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "temporary", data["type"])
			assert.Equal(t, "307", data["redirectType"])
			assert.Equal(t, "https://example.org", data["location"])
		})

	resp, err := client.Domains.AddURLForward(context.Background(), "example.com", &URLForward{
		Location:     "https://example.org",
		Type:         Temporary,
		RedirectType: Redirect307,
		IncludePath:  "no",
		Wildcard:     "no",
	})

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

// redirectType is optional and must be omitted when unset, so the API keeps
// applying its type-based default.
func TestDomains_AddDomainUrlForward_NoRedirectType(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/addUrlForward/example.com", "POST", "/domains/addUrlForward-success.http",
		func(data map[string]interface{}) {
			assert.NotContains(t, data, "redirectType")
		})

	_, err := client.Domains.AddURLForward(context.Background(), "example.com", &URLForward{
		Location:    "https://example.org",
		Type:        Permanent,
		IncludePath: "yes",
		Wildcard:    "no",
	})

	require.NoError(t, err)
}

// type is required by the API and redirectType only overrides it, so naming the
// exact code alone must still send a consistent pair.
func TestDomainsService_AddDomainUrlForward_DerivesTypeFromRedirectType(t *testing.T) {
	for _, tc := range []struct {
		redirect RedirectType
		want     ForwardType
	}{
		{Redirect301, Permanent},
		{Redirect302, Temporary},
		{Redirect307, Temporary},
		{RedirectMasked, Masked},
	} {
		t.Run(string(tc.redirect), func(t *testing.T) {
			setupMockServer(true)
			defer teardownMockServer()

			handleFixtureRequest(t, "/domain/addUrlForward/example.com", "POST", "/domains/addUrlForward-success.http",
				func(data map[string]interface{}) {
					assert.Equal(t, string(tc.want), data["type"])
					assert.Equal(t, string(tc.redirect), data["redirectType"])
				})

			forward := &URLForward{
				Location:     "https://example.org",
				RedirectType: tc.redirect,
				IncludePath:  "no",
				Wildcard:     "no",
			}

			_, err := client.Domains.AddURLForward(context.Background(), "example.com", forward)

			require.NoError(t, err)
			// The caller's struct is left alone.
			assert.Equal(t, ForwardType(""), forward.Type)
		})
	}
}

// An explicit Type wins: redirectType is the override, not the other way round.
func TestDomainsService_AddDomainUrlForward_KeepsExplicitType(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/addUrlForward/example.com", "POST", "/domains/addUrlForward-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, string(Temporary), data["type"])
			assert.Equal(t, string(Redirect307), data["redirectType"])
		})

	_, err := client.Domains.AddURLForward(context.Background(), "example.com", &URLForward{
		Location:     "https://example.org",
		Type:         Temporary,
		RedirectType: Redirect307,
		IncludePath:  "no",
		Wildcard:     "no",
	})

	require.NoError(t, err)
}

// includePath and wildcard are required, so an unset field is sent as "no"
// rather than as the empty string the API rejects.
func TestDomainsService_AddDomainUrlForward_DefaultsRequiredFlags(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/addUrlForward/example.com", "POST", "/domains/addUrlForward-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "no", data["includePath"])
			assert.Equal(t, "no", data["wildcard"])
		})

	forward := &URLForward{Location: "https://example.org", Type: Temporary}
	_, err := client.Domains.AddURLForward(context.Background(), "example.com", forward)

	require.NoError(t, err)
	// The caller's struct is left alone.
	assert.Empty(t, forward.IncludePath)
	assert.Empty(t, forward.Wildcard)
}
