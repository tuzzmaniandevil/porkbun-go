package porkbun

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

		require.NoError(t, err)
	})

	resp, err := client.Domains.ListDomains(context.Background(), &DomainListOptions{})

	require.NoError(t, err)
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

		require.NoError(t, err)
	})

	resp, err := client.Domains.ListDomains(context.Background(), &DomainListOptions{})

	// A date the API sends in an unexpected shape costs that one field, not the
	// response: failing here would abort the array decode and silently truncate
	// the page. Both dates in this fixture are malformed, so both read as zero
	// while every other field survives.
	require.NoError(t, err)
	require.Len(t, resp.Domains, 1, "the domain must not be dropped")
	assert.Equal(t, "borseth.ink", resp.Domains[0].Domain)
	assert.True(t, resp.Domains[0].CreateDate.IsZero(), "an unparsable date reads as the zero time")
	assert.True(t, resp.Domains[0].ExpireDate.IsZero())
}

func TestDomainsService_ListDomains_EmptyResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/listAll", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS","domains":[]}`)
	})

	resp, err := client.Domains.ListDomains(context.Background(), &DomainListOptions{})

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Empty(t, resp.Domains)
}

func TestDomainsService_ListDomains_MalformedResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/listAll", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS","domains":[{"domain":"example.com","createDate":"2020-`)
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
		_, _ = fmt.Fprint(w, `{
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

	options := &DomainListOptions{Start: Ptr(int64(1000))}
	resp, err := client.Domains.ListDomains(context.Background(), options)

	require.NoError(t, err)
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

	require.NoError(t, err)
	assert.True(t, domain.ExpireDate.IsZero(), "an unparsable date reads as the zero time")
	// Everything alongside it still decodes.
	assert.Equal(t, "example.com", domain.Domain)
	assert.Equal(t, 2024, domain.CreateDate.Year())
}

func TestDomainsService_ListDomains_PostFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/listAll", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"status":"ERROR","message":"Internal Server Error"}`)
	})

	_, err := client.Domains.ListDomains(context.Background(), &DomainListOptions{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500: Internal Server Error")
}

func TestDomains_ListDomains_Filters(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/listAll", "POST", "/domains/listAll-filtered.http",
		func(data map[string]interface{}) {
			assert.Equal(t, float64(0), data["start"])
			assert.Equal(t, "yes", data["includeLabels"])
			assert.Equal(t, "example.com", data["domain"])
			assert.Equal(t, "example", data["nameContains"])
			assert.Equal(t, float64(30), data["expiringWithinDays"])
			assert.Equal(t, []interface{}{"com", "io"}, data["tlds"])
			assert.Equal(t, "yes", data["autoRenew"])
			assert.Equal(t, "yes", data["apiAccess"])
			assert.Equal(t, "expire_date", data["sortName"])
			assert.Equal(t, "asc", data["sortDirection"])
		})

	days := int64(30)
	resp, err := client.Domains.ListDomains(context.Background(), &DomainListOptions{
		Start:              Ptr(int64(0)),
		IncludeLabels:      Yes,
		Domain:             String("example.com"),
		NameContains:       String("example"),
		ExpiringWithinDays: &days,
		TLDs:               []string{"com", "io"},
		AutoRenew:          Yes,
		APIAccess:          Yes,
		SortName:           DomainSortByExpireDate,
		SortDirection:      SortAscending,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Count)
	assert.Len(t, resp.Domains, 2)

	// Booleans arrive as strings on some fields and numbers on others.
	assert.Equal(t, BoolString(true), resp.Domains[0].SecurityLock)
	assert.Equal(t, BoolNumber(true), resp.Domains[0].AutoRenew)
	assert.Equal(t, BoolNumber(true), resp.Domains[0].APIAccess)
	assert.Equal(t, BoolNumber(false), resp.Domains[0].NotLocal)
	assert.Equal(t, BoolString(false), resp.Domains[1].SecurityLock)
	assert.Equal(t, BoolNumber(false), resp.Domains[1].APIAccess)
	assert.Equal(t, BoolNumber(true), resp.Domains[1].NotLocal)
}

// The dates use the API's own format rather than RFC 3339, so a Domain that goes
// out again has to carry them in the form it received them.
func TestDomain_RoundTripsThroughJSON(t *testing.T) {
	const in = `{"domain":"example.com","status":"ACTIVE","tld":"com",` +
		`"createDate":"2021-01-15 10:00:00","expireDate":"2027-01-15 10:00:00",` +
		`"securityLock":"1","whoisPrivacy":"1","autoRenew":1,"apiAccess":1,"notLocal":0}`

	var domain Domain
	require.NoError(t, json.Unmarshal([]byte(in), &domain))

	out, err := json.Marshal(domain)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))

	assert.Equal(t, "2021-01-15 10:00:00", got["createDate"])
	assert.Equal(t, "2027-01-15 10:00:00", got["expireDate"])

	// And the result decodes back into an identical Domain.
	var again Domain
	require.NoError(t, json.Unmarshal(out, &again))
	assert.Equal(t, domain, again)
}

// A zero date stays empty rather than becoming year one.
func TestDomain_MarshalJSON_ZeroDates(t *testing.T) {
	out, err := json.Marshal(Domain{Domain: "example.com"})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))

	assert.Equal(t, "", got["createDate"])
	assert.Equal(t, "", got["expireDate"])
}

func TestDomains_GetDomain(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/get/tuzzatech.co", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "yes", r.URL.Query().Get("includeLabels"))

		fixtureHandler(t, "GET", "/domains/get-success.http", nil)(w, r)
	})

	resp, err := client.Domains.GetDomain(context.Background(), "tuzzatech.co", &GetDomainOptions{IncludeLabels: Yes})

	require.NoError(t, err)
	assert.Equal(t, "tuzzatech.co", resp.Domain.Domain)
	assert.Equal(t, "ACTIVE", resp.Domain.Status)
	assert.Equal(t, "co", resp.Domain.TLD)
	assert.Equal(t, 2019, resp.Domain.CreateDate.Year())
	assert.Equal(t, 2026, resp.Domain.ExpireDate.Year())
	assert.Equal(t, BoolNumber(true), resp.Domain.APIAccess)
	assert.Len(t, resp.Domain.Labels, 1)
	assert.Equal(t, "cool", resp.Domain.Labels[0].Title)
	assert.NotEmpty(t, resp.Domain.Labels[0].ID)
	assert.NotEmpty(t, resp.Domain.Labels[0].Color)
}

// includeLabels is optional, so a nil value must not appear in the query.
func TestDomains_GetDomain_NoLabels(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/get/tuzzatech.co", func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.RawQuery)

		fixtureHandler(t, "GET", "/domains/get-success.http", nil)(w, r)
	})

	resp, err := client.Domains.GetDomain(context.Background(), "tuzzatech.co", nil)

	require.NoError(t, err)
	assert.Equal(t, "tuzzatech.co", resp.Domain.Domain)
}

func TestDomains_GetDomain_NotFound(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/get/nope.com", "GET", "/domains/get-notfound.http")

	_, err := client.Domains.GetDomain(context.Background(), "nope.com", nil)

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrCodeDomainNotFound, errResponse.Code)
	assert.Equal(t, http.StatusNotFound, errResponse.HTTPResponse.StatusCode)
}

func TestDomains_UpdateAutoRenew(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/updateAutoRenew/example.com", "POST", "/domains/updateAutoRenew-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "on", data["status"])
			assert.Equal(t, []interface{}{"other.com"}, data["domains"])
		})

	resp, err := client.Domains.UpdateAutoRenew(context.Background(), AutoRenewOn, []string{"example.com", "other.com"})

	require.NoError(t, err)
	assert.Len(t, resp.Results, 2)
	assert.Equal(t, "SUCCESS", resp.Results["example.com"].Status)
	assert.Equal(t, "Auto-renew enabled.", resp.Results["other.com"].Message)
}

// A single domain in the path needs no domains array.
func TestDomains_UpdateAutoRenew_SingleDomain(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/updateAutoRenew/example.com", "POST", "/domains/updateAutoRenew-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "off", data["status"])
			assert.NotContains(t, data, "domains")
		})

	_, err := client.Domains.UpdateAutoRenew(context.Background(), AutoRenewOff, []string{"example.com"})

	require.NoError(t, err)
}

// The API takes one domain in the path and the rest in the body, and combines
// them, so a caller passes one list and the SDK splits it.
func TestDomains_UpdateAutoRenew_Bulk(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/updateAutoRenew/example.com", "POST", "/domains/updateAutoRenew-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, []interface{}{"other.com"}, data["domains"],
				"the first domain travels in the path, the remainder in the body")
		})

	_, err := client.Domains.UpdateAutoRenew(context.Background(), AutoRenewOn, []string{"example.com", "other.com"})

	require.NoError(t, err)
}

// An empty list is a caller error, caught before a request goes out.
func TestDomains_UpdateAutoRenew_NoDomains(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no request should be sent: %s %s", r.Method, r.URL.Path)
	})

	resp, err := client.Domains.UpdateAutoRenew(context.Background(), AutoRenewOn, nil)

	assert.ErrorContains(t, err, "at least one domain")
	assert.NotNil(t, resp)
}

// Per-domain failures come back inside a successful response.
func TestDomains_UpdateAutoRenew_PartialFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/updateAutoRenew/example.com", "POST", "/domains/updateAutoRenew-partial.http")

	resp, err := client.Domains.UpdateAutoRenew(context.Background(), AutoRenewOn, []string{"example.com", "notmine.com"})

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Results["example.com"].Status)
	assert.Equal(t, "ERROR", resp.Results["notmine.com"].Status)
	assert.Equal(t, "Domain is not in your account.", resp.Results["notmine.com"].Message)
}

func TestDomains_UpdateAutoRenew_Error(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/updateAutoRenew/example.com", "POST", "/domains/updateAutoRenew-error.http")

	_, err := client.Domains.UpdateAutoRenew(context.Background(), "maybe", []string{"example.com"})

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrorCode("INVALID_STATUS"), errResponse.Code)
}

// Domain.APIAccess is a Bool, so omitempty would drop it for a domain that is not
// opted in, losing a field that securityLock, whoisPrivacy and notLocal keep.
func TestDomain_ApiAccessSurvivesARoundTrip(t *testing.T) {
	const raw = `{"domain":"example.com","status":"ACTIVE","tld":"com",` +
		`"createDate":"2024-01-02 03:04:05","expireDate":"2025-01-02 03:04:05",` +
		`"securityLock":"1","whoisPrivacy":"0","autoRenew":1,"apiAccess":0,"notLocal":0}`

	var domain Domain
	require.NoError(t, json.Unmarshal([]byte(raw), &domain))
	assert.Equal(t, BoolNumber(false), domain.APIAccess)

	out, err := json.Marshal(domain)
	require.NoError(t, err)

	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(out, &fields))
	assert.Contains(t, fields, "apiAccess", "apiAccess must survive even when false")

	// The dates keep the format the API uses, not RFC 3339.
	assert.JSONEq(t, `"2024-01-02 03:04:05"`, string(fields["createDate"]))
}
