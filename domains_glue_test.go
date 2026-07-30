package porkbun

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomains_GetGlueRecords(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/getGlue/example.com", "POST", "/domains/getGlue-success.http")

	resp, err := client.Domains.GetGlueRecords(context.Background(), "example.com")

	require.NoError(t, err)
	assert.Len(t, resp.Hosts, 2)
	assert.Equal(t, "ns1.example.com", resp.Hosts[0].Host)
	assert.Equal(t, []string{"1.2.3.4"}, resp.Hosts[0].IPs.V4)
	assert.Equal(t, []string{"2001:db8::1"}, resp.Hosts[0].IPs.V6)
	assert.Equal(t, "ns2.example.com", resp.Hosts[1].Host)
	assert.Equal(t, []string{"5.6.7.8"}, resp.Hosts[1].IPs.V4)
	assert.Empty(t, resp.Hosts[1].IPs.V6)
}

// hosts is a non-nullable array in the specification, so a domain with no glue
// records comes back as [].
func TestDomains_GetGlueRecords_Empty(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/getGlue/example.com", "POST", "/domains/getGlue-empty.http")

	resp, err := client.Domains.GetGlueRecords(context.Background(), "example.com")

	require.NoError(t, err)
	assert.Empty(t, resp.Hosts)
}

func TestDomains_CreateGlueRecord(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/createGlue/example.com/ns1", "POST", "/domains/createGlue-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, []interface{}{"1.2.3.4", "2001:db8::1"}, data["ips"])
		})

	resp, err := client.Domains.CreateGlueRecord(context.Background(), "example.com", "ns1",
		[]string{"1.2.3.4", "2001:db8::1"})

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

func TestDomains_UpdateGlueRecord(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/updateGlue/example.com/ns1", "POST", "/domains/updateGlue-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, []interface{}{"5.6.7.8"}, data["ips"])
		})

	resp, err := client.Domains.UpdateGlueRecord(context.Background(), "example.com", "ns1", []string{"5.6.7.8"})

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

// Delete takes no addresses, so the ips field must be omitted rather than sent as null.
func TestDomains_DeleteGlueRecord(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/deleteGlue/example.com/ns1", "POST", "/domains/deleteGlue-success.http",
		func(data map[string]interface{}) {
			assert.NotContains(t, data, "ips")
		})

	resp, err := client.Domains.DeleteGlueRecord(context.Background(), "example.com", "ns1")

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

func TestDomains_CreateGlueRecord_Error(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/createGlue/example.com/ns1", "POST", "/domains/glue-error.http")

	_, err := client.Domains.CreateGlueRecord(context.Background(), "example.com", "ns1", []string{"nope"})

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrorCode("INVALID_IP"), errResponse.Code)
}

func TestGlueRecord_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr assert.ErrorAssertionFunc
	}{
		{name: "tuple", data: []byte(`["ns1.example.com",{"v4":["1.2.3.4"],"v6":[]}]`), wantErr: assert.NoError},
		{name: "too few elements", data: []byte(`["ns1.example.com"]`), wantErr: assert.Error},
		{name: "too many elements", data: []byte(`["a",{},"b"]`), wantErr: assert.Error},
		{name: "not an array", data: []byte(`{"host":"ns1.example.com"}`), wantErr: assert.Error},
		{name: "bad hostname type", data: []byte(`[42,{"v4":[]}]`), wantErr: assert.Error},
		{name: "bad ips type", data: []byte(`["ns1.example.com","1.2.3.4"]`), wantErr: assert.Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var record GlueRecord
			tt.wantErr(t, json.Unmarshal(tt.data, &record))
		})
	}
}

// GlueRecord decodes a [hostname, ips] tuple, so it has to encode one too: a
// response re-marshalled with the Go field names is not something the API will
// accept back, and neither will this package's own UnmarshalJSON.
func TestGlueRecord_RoundTripsThroughJSON(t *testing.T) {
	const raw = `{"status":"SUCCESS","hosts":[["ns1.example.com",{"v4":["1.2.3.4"],"v6":["2001:db8::1"]}]]}`

	var resp GetGlueRecordsResponse
	require.NoError(t, json.Unmarshal([]byte(raw), &resp))
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, "ns1.example.com", resp.Hosts[0].Host)

	out, err := json.Marshal(resp.Hosts)
	require.NoError(t, err)
	assert.JSONEq(t, `[["ns1.example.com",{"v4":["1.2.3.4"],"v6":["2001:db8::1"]}]]`, string(out))

	// And the encoded form decodes back into the same value.
	var back []GlueRecord
	require.NoError(t, json.Unmarshal(out, &back))
	assert.Equal(t, resp.Hosts, back)
}
