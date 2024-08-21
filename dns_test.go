package porkbun

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDnsService_DnsPath(t *testing.T) {
	assert.Equal(t, "/dns/retrieve", dnsPath("retrieve"))
	assert.Equal(t, "/dns/retrieve/example.com", dnsPath("retrieve", "example.com"))
	assert.Equal(t, "/dns/retrieveByNameType/example.com", dnsPath("retrieveByNameType", "example.com", nil))
	assert.Equal(t, "/dns/retrieveByNameType/example.com/ALIAS", dnsPath("retrieveByNameType", "example.com", ALIAS))
}

func TestDnsService_DnsRecordType(t *testing.T) {
	a := DnsRecordType("A")
	assert.True(t, a.IsValid())

	var exp *string
	assert.IsType(t, exp, a.ToPtr())

	invalid := DnsRecordType("INVALID")
	assert.False(t, invalid.IsValid())
}

func TestDnsService_GetRecords(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/retrieve/example.com", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/dns/retrieveByDomain/success.http")
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

	resp, err := client.Dns.GetRecords(context.Background(), "example.com", nil)

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Len(t, resp.Records, 6)
}

func TestDnsService_GetRecordsId(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/retrieve/example.com/421766139", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/dns/retrieveByDomainId/success.http")
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

	recordId := int64(421766139)
	resp, err := client.Dns.GetRecords(context.Background(), "example.com", &recordId)

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Len(t, resp.Records, 1)
}

func TestDnsService_GetRecordsByType(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/retrieveByNameType/example.com/ALIAS", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/dns/getRecordsByType/success.http")
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

	resp, err := client.Dns.GetRecordsByType(context.Background(), "example.com", "ALIAS", nil)

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Len(t, resp.Records, 1)
}

func TestDnsService_GetRecordsByTypeSubdomain(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/retrieveByNameType/example.com/ALIAS/www", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/dns/getRecordsByTypeSubdomain/success.http")
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

	resp, err := client.Dns.GetRecordsByType(context.Background(), "example.com", ALIAS, String("www"))

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Len(t, resp.Records, 1)
}

func TestDnsService_CreateRecord(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/create/example.com", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/dns/createRecord/success.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)
		testCredentials(t, r)

		expectedBody := map[string]interface{}{
			"apikey":       "1234",
			"secretapikey": "5678",
			"name":         "",
			"type":         "A",
			"content":      "192.0.2.1",
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

	resp, err := client.Dns.CreateRecord(context.Background(), "example.com", &DnsRecord{
		Name:    "",
		Type:    A,
		Content: "192.0.2.1",
	})

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Equal(t, resp.ID, int64(1234))
}

func TestDnsService_EditRecord(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/edit/example.com/1234", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/dns/editRecord/success.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)
		testCredentials(t, r)

		expectedBody := map[string]interface{}{
			"apikey":       "1234",
			"secretapikey": "5678",
			"name":         "secret",
			"type":         "ALIAS",
			"content":      "1.1.1.1",
			"ttl":          "300",
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

	resp, err := client.Dns.EditRecord(context.Background(), "example.com", int64(1234), &EditRecord{
		Name:    "secret",
		Type:    ALIAS,
		Content: "1.1.1.1",
		TTL:     "300",
	})

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

func TestDnsService_EditRecordByType(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/editByNameType/example.com/ALIAS/secret", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/dns/editRecordByType/success.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)
		testCredentials(t, r)

		expectedBody := map[string]interface{}{
			"apikey":       "1234",
			"secretapikey": "5678",
			"content":      "pixie.porkbun.com",
			"ttl":          "600",
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

	resp, err := client.Dns.EditRecordByType(context.Background(), "example.com", ALIAS, String("secret"), &EditTypeRecord{
		Content: "pixie.porkbun.com",
		TTL:     "600",
	})

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

func TestDnsService_DeleteRecord(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/delete/example.com/1234", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/dns/deleteRecord/success.http")
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

	resp, err := client.Dns.DeleteRecord(context.Background(), "example.com", int64(1234))

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

func TestDnsService_DeleteRecordByType(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/deleteByNameType/example.com/A", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/dns/deleteRecord/success.http")
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

	resp, err := client.Dns.DeleteRecordByType(context.Background(), "example.com", A, nil)

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

func TestDnsService_GetRecords_EmptyResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/retrieve/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"SUCCESS","records":[]}`)
	})

	resp, err := client.Dns.GetRecords(context.Background(), "example.com", nil)

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Empty(t, resp.Records)
}

func TestDnsService_GetRecords_MalformedResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/retrieve/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"SUCCESS","records":[{"id":"1234","type":"A","content":"192.0.2.1"}`)
	})

	_, err := client.Dns.GetRecords(context.Background(), "example.com", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
}

func TestDnsService_CreateRecord_InvalidType(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/create/example.com", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/dns/createRecord/invalidType.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)
		testCredentials(t, r)

		expectedBody := map[string]interface{}{
			"apikey":       "1234",
			"secretapikey": "5678",
			"name":         "",
			"type":         "INVALID",
			"content":      "192.0.2.1",
		}
		testRequestJSON(t, r, expectedBody)

		w.WriteHeader(httpResponse.StatusCode)
		_, err := io.Copy(w, httpResponse.Body)

		assert.NoError(t, err)
	})

	_, err := client.Dns.CreateRecord(context.Background(), "example.com", &DnsRecord{
		Name:    "",
		Type:    DnsRecordType("INVALID"),
		Content: "192.0.2.1",
	})

	assert.Error(t, err)
}

func TestDnsService_GetRecordsByType_PostFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/retrieveByNameType/example.com/ALIAS", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"status":"ERROR","message":"Internal Server Error"}`)
	})

	_, err := client.Dns.GetRecordsByType(context.Background(), "example.com", ALIAS, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500 Internal Server Error")
}

func TestDnsService_EditRecordByType_PostFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/editByNameType/example.com/ALIAS/secret", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"status":"ERROR","message":"Internal Server Error"}`)
	})

	_, err := client.Dns.EditRecordByType(context.Background(), "example.com", ALIAS, String("secret"), &EditTypeRecord{
		Content: "pixie.porkbun.com",
		TTL:     "600",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500 Internal Server Error")
}

func TestDnsService_DeleteRecordByType_PostFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/deleteByNameType/example.com/A", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"status":"ERROR","message":"Internal Server Error"}`)
	})

	_, err := client.Dns.DeleteRecordByType(context.Background(), "example.com", A, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500 Internal Server Error")
}
