package porkbun

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDomainsService_ListDomains(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/listAll", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/domains/listAll-success.http")
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

	resp, err := client.Domains.ListDomains(context.Background(), &DomainListOptions{})

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.NotEmpty(t, resp.Domains)

	testDomain := resp.Domains[0]
	assert.Equal(t, "borseth.ink", testDomain.Domain)
	assert.True(t, bool(testDomain.SecurityLock))
	assert.True(t, bool(testDomain.WhoisPrivacy))
	assert.False(t, bool(testDomain.AutoRenew))
	assert.False(t, bool(testDomain.NotLocal))
}

func TestDomainsService_ListDomains_InvalidTime(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/listAll", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/domains/listAll-invalid-time.http")
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

	_, err := client.Domains.ListDomains(context.Background(), &DomainListOptions{})

	assert.Error(t, err)
}

func TestDomainsService_ListDomains_EmptyResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/listAll", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"SUCCESS","domains":[]}`)
	})

	resp, err := client.Domains.ListDomains(context.Background(), &DomainListOptions{})

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Empty(t, resp.Domains)
}

func TestDomainsService_ListDomains_MalformedResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/listAll", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"SUCCESS","domains":[{"domain":"example.com","createDate":"2020-`)
	})

	_, err := client.Domains.ListDomains(context.Background(), &DomainListOptions{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
}

func TestDomainsService_ListDomains_Pagination(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/listAll", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
            "status":"SUCCESS",
            "domains":[
                {
                    "domain":"page1.com",
                    "status": "ACTIVE",
                    "tld": "com",
                    "createDate": "2023-01-01 12:00:00",
                    "expireDate": "2024-01-01 12:00:00",
                    "securityLock": "1",
                    "whoisPrivacy": "1",
                    "autoRenew": 0,
                    "notLocal": 0
                }
            ]
        }`)
	})

	options := &DomainListOptions{Start: String("1000")}
	resp, err := client.Domains.ListDomains(context.Background(), options)

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Len(t, resp.Domains, 1)
	assert.Equal(t, "page1.com", resp.Domains[0].Domain)
}

func TestDomain_UnmarshalJSON_InvalidExpireDate(t *testing.T) {
	data := []byte(`{
        "domain": "example.com",
        "status": "active",
        "tld": "com",
        "createDate": "2024-01-01 12:00:00",
        "expireDate": "invalid-date",
        "securityLock": "1",
        "whoisPrivacy": "1",
        "autoRenew": 0,
        "notLocal": 0
    }`)

	var domain Domain
	err := json.Unmarshal(data, &domain)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing ExpireDate")
}

func TestDomainsService_ListDomains_PostFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/listAll", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"status":"ERROR","message":"Internal Server Error"}`)
	})

	_, err := client.Domains.ListDomains(context.Background(), &DomainListOptions{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500 Internal Server Error")
}
