package porkbun

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
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

		assert.NoError(t, err)
	})

	resp, err := client.Domains.GetDomainURLForwarding(context.Background(), "example.com")

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.NotEmpty(t, resp.Forwards)

	forward := resp.Forwards[0]
	assert.Equal(t, "22049216", forward.Id)
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

		assert.NoError(t, err)
	})

	resp, err := client.Domains.AddDomainUrlForward(context.Background(), "example.com", &UrlForward{
		Location:    "https://porkbun.com",
		Type:        Temporary,
		IncludePath: "yes",
		Wildcard:    "yes",
	})

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
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

		assert.NoError(t, err)
	})

	_, err := client.Domains.AddDomainUrlForward(context.Background(), "unknowndomain.com", &UrlForward{
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

		assert.NoError(t, err)
	})

	resp, err := client.Domains.DeleteDomainUrlForward(context.Background(), "example.com", "12345")

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

func TestDomainsService_GetDomainURLForwarding_InvalidResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/getUrlForwarding/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"SUCCESS","forwards":"invalid"}`)
	})

	_, err := client.Domains.GetDomainURLForwarding(context.Background(), "example.com")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "json: cannot unmarshal")
}

func TestDomainsService_GetDomainURLForwarding_EmptyResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/getUrlForwarding/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"SUCCESS","forwards":[]}`)
	})

	resp, err := client.Domains.GetDomainURLForwarding(context.Background(), "example.com")

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Empty(t, resp.Forwards)
}

func TestDomainsService_GetDomainURLForwarding_PartialResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/getUrlForwarding/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
            "status":"SUCCESS",
            "forwards":[{
                "id":"22049216",
                "type":"temporary",
                "location":"https://porkbun.com"
            }]
        }`)
	})

	resp, err := client.Domains.GetDomainURLForwarding(context.Background(), "example.com")

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Len(t, resp.Forwards, 1)
	assert.Equal(t, "22049216", resp.Forwards[0].Id)
	assert.Equal(t, Temporary, resp.Forwards[0].Type)
	assert.Equal(t, "https://porkbun.com", resp.Forwards[0].Location)
}

func TestDomainsService_DeleteDomainUrlForward_PostFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/deleteUrlForward/example.com/12345", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"status":"ERROR","message":"Internal Server Error"}`)
	})

	_, err := client.Domains.DeleteDomainUrlForward(context.Background(), "example.com", "12345")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500 Internal Server Error")
}
