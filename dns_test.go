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

func TestDnsService_DnsPath(t *testing.T) {
	assert.Equal(t, "/dns/retrieve", apiPath("dns", "retrieve"))
	assert.Equal(t, "/dns/retrieve/example.com", apiPath("dns", "retrieve", "example.com"))
	assert.Equal(t, "/dns/retrieveByNameType/example.com", apiPath("dns", "retrieveByNameType", "example.com", nil))
	assert.Equal(t, "/dns/retrieveByNameType/example.com/ALIAS", apiPath("dns", "retrieveByNameType", "example.com", ALIAS))
}

func TestDnsService_DnsRecordType(t *testing.T) {
	assert.True(t, DNSRecordType("A").IsValid())

	invalid := DNSRecordType("INVALID")
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

		require.NoError(t, err)
	})

	resp, err := client.DNS.GetRecords(context.Background(), "example.com", nil)

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Len(t, resp.Records, 6)

	// Retrieval quotes the numbers and sends prio as null on record types that
	// have none, so the numeric fields have to accept both.
	assert.Equal(t, FlexInt64(12345), resp.Records[0].ID)
	assert.Equal(t, FlexInt64(600), resp.Records[0].TTL)
	assert.Equal(t, FlexInt64(0), resp.Records[0].Prio)
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

		require.NoError(t, err)
	})

	recordID := int64(421766139)
	resp, err := client.DNS.GetRecords(context.Background(), "example.com", &recordID)

	require.NoError(t, err)
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

		require.NoError(t, err)
	})

	resp, err := client.DNS.GetRecordsByType(context.Background(), "example.com", "ALIAS", nil)

	require.NoError(t, err)
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

		require.NoError(t, err)
	})

	resp, err := client.DNS.GetRecordsByType(context.Background(), "example.com", ALIAS, String("www"))

	require.NoError(t, err)
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

		require.NoError(t, err)
	})

	resp, err := client.DNS.CreateRecord(context.Background(), "example.com", &DNSRecord{
		Name:    "",
		Type:    A,
		Content: "192.0.2.1",
	})

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Equal(t, int64(1234), resp.ID.Int64())
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
			"ttl":          float64(300),
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

	resp, err := client.DNS.EditRecord(context.Background(), "example.com", int64(1234), &EditRecord{
		Name:    "secret",
		Type:    ALIAS,
		Content: "1.1.1.1",
		TTL:     300,
	})

	require.NoError(t, err)
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
			"ttl":          float64(600),
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

	resp, err := client.DNS.EditRecordByType(context.Background(), "example.com", ALIAS, String("secret"), &EditTypeRecord{
		Content: "pixie.porkbun.com",
		TTL:     600,
	})

	require.NoError(t, err)
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

		require.NoError(t, err)
	})

	resp, err := client.DNS.DeleteRecord(context.Background(), "example.com", int64(1234))

	require.NoError(t, err)
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

		require.NoError(t, err)
	})

	resp, err := client.DNS.DeleteRecordByType(context.Background(), "example.com", A, nil)

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

func TestDnsService_GetRecords_EmptyResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/retrieve/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS","records":[]}`)
	})

	resp, err := client.DNS.GetRecords(context.Background(), "example.com", nil)

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Empty(t, resp.Records)
}

func TestDnsService_GetRecords_MalformedResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/retrieve/example.com", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS","records":[{"id":"1234","type":"A","content":"192.0.2.1"}`)
	})

	_, err := client.DNS.GetRecords(context.Background(), "example.com", nil)

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

		require.NoError(t, err)
	})

	_, err := client.DNS.CreateRecord(context.Background(), "example.com", &DNSRecord{
		Name:    "",
		Type:    DNSRecordType("INVALID"),
		Content: "192.0.2.1",
	})

	assert.Error(t, err)
}

func TestDnsService_GetRecordsByType_PostFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/retrieveByNameType/example.com/ALIAS", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"status":"ERROR","message":"Internal Server Error"}`)
	})

	_, err := client.DNS.GetRecordsByType(context.Background(), "example.com", ALIAS, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500: Internal Server Error")
}

func TestDnsService_EditRecordByType_PostFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/editByNameType/example.com/ALIAS/secret", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"status":"ERROR","message":"Internal Server Error"}`)
	})

	_, err := client.DNS.EditRecordByType(context.Background(), "example.com", ALIAS, String("secret"), &EditTypeRecord{
		Content: "pixie.porkbun.com",
		TTL:     600,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500: Internal Server Error")
}

func TestDnsService_DeleteRecordByType_PostFailure(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/dns/deleteByNameType/example.com/A", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"status":"ERROR","message":"Internal Server Error"}`)
	})

	_, err := client.DNS.DeleteRecordByType(context.Background(), "example.com", A, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500: Internal Server Error")
}

// A new record id arrives as a JSON string on this endpoint and still decodes
// into the int64 ID field.
func TestDns_CreateRecord_StringID(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/dns/create/example.com", "POST", "/dns/createRecord/stringId.http")

	resp, err := client.DNS.CreateRecord(context.Background(), "example.com", &DNSRecord{
		Type:    A,
		Content: "1.2.3.4",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(106926659), resp.ID.Int64())
}

func TestDns_GetRecords_Cloudflare(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/dns/retrieve/example.com", "POST", "/dns/retrieveByDomain/cloudflare.http")

	resp, err := client.DNS.GetRecords(context.Background(), "example.com", nil)

	require.NoError(t, err)
	assert.Equal(t, CloudflareEnabled, resp.Cloudflare)
	assert.Len(t, resp.Records, 1)
	assert.Equal(t, "apex", resp.Records[0].Notes)
}

func TestDns_CreateRecord_DryRun(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/dns/create/example.com", "POST", "/dns/createRecord/dryRun.http",
		func(data map[string]interface{}) {
			assert.Equal(t, true, data["dryRun"])
		})

	resp, err := client.DNS.CreateRecord(context.Background(), "example.com", &DNSRecord{
		Type:    A,
		Content: "1.2.3.4",
	}, WithDryRun())

	require.NoError(t, err)
	// A dry run is only distinguishable from a real success through these fields,
	// without them a caller cannot tell whether the record was actually created.
	assert.True(t, resp.DryRun)
	assert.True(t, resp.WouldSucceed)
	assert.Contains(t, resp.Message, "Dry run")
	assert.Equal(t, int64(0), resp.ID.Int64())
}

func TestDns_DeleteRecord_DryRun(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/dns/delete/example.com/106926659", "POST", "/dns/deleteRecord/dryRun.http",
		func(data map[string]interface{}) {
			assert.Equal(t, true, data["dryRun"])
		})

	resp, err := client.DNS.DeleteRecord(context.Background(), "example.com", 106926659, WithDryRun())

	require.NoError(t, err)
	assert.True(t, resp.DryRun)
	assert.True(t, resp.WouldSucceed)
	assert.Contains(t, resp.Message, "Dry run")
}

// Notes are only cleared by an explicit empty string; nil leaves them unchanged.
func TestDns_EditRecord_Notes(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/dns/edit/example.com/106926659", "POST", "/dns/editRecord/success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "", data["notes"])
		})

	_, err := client.DNS.EditRecord(context.Background(), "example.com", 106926659, &EditRecord{
		Type:    A,
		Content: "1.2.3.4",
		Notes:   String(""),
	})

	require.NoError(t, err)
}

func TestDns_EditRecord_NotesOmitted(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/dns/edit/example.com/106926659", "POST", "/dns/editRecord/success.http",
		func(data map[string]interface{}) {
			assert.NotContains(t, data, "notes")
		})

	_, err := client.DNS.EditRecord(context.Background(), "example.com", 106926659, &EditRecord{
		Type:    A,
		Content: "1.2.3.4",
	})

	require.NoError(t, err)
}

func TestDns_EditRecordByType_Notes(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/dns/editByNameType/example.com/A/www", "POST", "/dns/editRecordByType/success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "rotated", data["notes"])
		})

	_, err := client.DNS.EditRecordByType(context.Background(), "example.com", A, String("www"), &EditTypeRecord{
		Content: "1.2.3.4",
		Notes:   String("rotated"),
	})

	require.NoError(t, err)
}

func TestDnsRecordType_SSHFP(t *testing.T) {
	assert.True(t, SSHFP.IsValid())
	assert.Equal(t, "SSHFP", SSHFP.String())
}

func TestUnqualifyName(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   string
	}{
		{"www.example.com", "example.com", "www"},                 // the form GetRecords returns
		{"example.com", "example.com", ""},                        // the root record
		{"*.example.com", "example.com", "*"},                     // wildcard
		{"a.b.example.com", "example.com", "a.b"},                 // nested subdomain
		{"WWW.Example.COM", "example.com", "WWW"},                 // DNS names are case-insensitive
		{"www", "example.com", "www"},                             // already bare, left alone
		{"", "example.com", ""},                                   // root, written the other way
		{"example.com.example.com", "example.com", "example.com"}, // only one suffix comes off
		{"notexample.com", "example.com", "notexample.com"},       // suffix without the dot is not a match
	}

	for _, tt := range tests {
		t.Run(tt.name+"/"+tt.domain, func(t *testing.T) {
			assert.Equal(t, tt.want, unqualifyName(tt.name, tt.domain))
		})
	}
}

// A record read back from GetRecords carries a fully qualified name, which the
// create endpoint would qualify a second time.
func TestDns_CreateRecord_UnqualifiesName(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/dns/create/example.com", "POST", "/dns/createRecord/success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "www", data["name"])
		})

	record := &DNSRecord{ID: 1234, Name: "www.example.com", Type: A, Content: "192.0.2.1"}
	_, err := client.DNS.CreateRecord(context.Background(), "example.com", record)

	require.NoError(t, err)
	// The caller's struct is left alone.
	assert.Equal(t, "www.example.com", record.Name)
	assert.Equal(t, int64(1234), record.ID.Int64())
}

func TestDns_EditRecord_UnqualifiesName(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/dns/edit/example.com/1234", "POST", "/dns/editRecord/success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "", data["name"])
		})

	record := &EditRecord{Name: "example.com", Type: A, Content: "192.0.2.1"}
	_, err := client.DNS.EditRecord(context.Background(), "example.com", 1234, record)

	require.NoError(t, err)
	assert.Equal(t, "example.com", record.Name)
}

// The byType endpoints carry the subdomain in the path rather than the body, and
// unqualify it as CreateRecord and EditRecord do for the record name: a retrieved
// name passed through untouched addresses /A/www.example.com, which matches no
// record, so a delete would remove nothing and report success.
//
// Each handler is registered on the bare path, so a regression fails as a 404
// rather than as a wrong assertion.
func TestDns_ByType_UnqualifiesSubdomain(t *testing.T) {
	qualified := Ptr("www.example.com")

	t.Run("GetRecordsByType", func(t *testing.T) {
		setupMockServer(true)
		defer teardownMockServer()

		handleFixtureRequest(t, "/dns/retrieveByNameType/example.com/A/www", "POST",
			"/dns/getRecordsByTypeSubdomain/success.http", nil)

		_, err := client.DNS.GetRecordsByType(context.Background(), "example.com", A, qualified)
		require.NoError(t, err)
	})

	t.Run("EditRecordByType", func(t *testing.T) {
		setupMockServer(true)
		defer teardownMockServer()

		handleFixtureRequest(t, "/dns/editByNameType/example.com/A/www", "POST",
			"/dns/editRecordByType/success.http", nil)

		_, err := client.DNS.EditRecordByType(context.Background(), "example.com", A, qualified,
			&EditTypeRecord{Content: "192.0.2.1"})
		require.NoError(t, err)
	})

	t.Run("DeleteRecordByType", func(t *testing.T) {
		setupMockServer(true)
		defer teardownMockServer()

		handleFixtureRequest(t, "/dns/deleteByNameType/example.com/A/www", "POST",
			"/dns/deleteRecord/success.http", nil)

		_, err := client.DNS.DeleteRecordByType(context.Background(), "example.com", A, qualified)
		require.NoError(t, err)
	})

	// The caller's string is never written through.
	assert.Equal(t, "www.example.com", *qualified)
}

// ListRecords and GetRecord are the two operations GetRecords used to select
// between with a pointer; each must hit its own path.
func TestDNS_ListRecordsAndGetRecord(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/dns/retrieve/example.com", "POST", "/dns/retrieveByDomain/success.http")
	handleFixture(t, "/dns/retrieve/example.com/421766139", "POST", "/dns/retrieveByDomainId/success.http")

	t.Run("ListRecords", func(t *testing.T) {
		resp, err := client.DNS.ListRecords(context.Background(), "example.com")

		require.NoError(t, err)
		assert.Equal(t, "SUCCESS", resp.Status)
		assert.NotEmpty(t, resp.Records)
	})

	t.Run("GetRecord", func(t *testing.T) {
		resp, err := client.DNS.GetRecord(context.Background(), "example.com", 421766139)

		require.NoError(t, err)
		assert.Equal(t, "SUCCESS", resp.Status)
		assert.Len(t, resp.Records, 1)
	})
}

// EditRecordResponse and DeleteRecordResponse embed DryRunResult, but only the
// create and delete-by-id paths ever decoded one. The API documents dryRun on all
// five DNS writes.
func TestDNS_DryRunVerdictDecodesOnEveryWrite(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/dns/edit/example.com/1234", "POST", "/dns/editRecord/dryRun.http")
	handleFixture(t, "/dns/editByNameType/example.com/A/www", "POST", "/dns/editRecord/dryRun.http")
	handleFixture(t, "/dns/deleteByNameType/example.com/A/www", "POST", "/dns/editRecord/dryRun.http")

	ctx := context.Background()

	t.Run("EditRecord", func(t *testing.T) {
		resp, err := client.DNS.EditRecord(ctx, "example.com", 1234,
			&EditRecord{Name: "www", Type: A, Content: "1.2.3.4"}, WithDryRun())

		require.NoError(t, err)
		assert.True(t, resp.DryRun)
		assert.True(t, resp.WouldSucceed)
	})

	t.Run("EditRecordByType", func(t *testing.T) {
		resp, err := client.DNS.EditRecordByType(ctx, "example.com", A, String("www"),
			&EditTypeRecord{Content: "1.2.3.4"}, WithDryRun())

		require.NoError(t, err)
		assert.True(t, resp.DryRun)
		assert.True(t, resp.WouldSucceed)
	})

	t.Run("DeleteRecordByType", func(t *testing.T) {
		resp, err := client.DNS.DeleteRecordByType(ctx, "example.com", A, String("www"), WithDryRun())

		require.NoError(t, err)
		assert.True(t, resp.DryRun)
		assert.True(t, resp.WouldSucceed)
	})
}
