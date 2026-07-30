package porkbun

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every map-typed field arrives as [] when it has no entries, because the API's
// backend cannot tell an empty map from an empty list. Decoding that as an error
// would fail the ordinary cases: a domain without DNSSEC, a TLD without coupons.
func TestFlexMap_EmptyArrayDecodesAsEmptyMap(t *testing.T) {
	t.Run("pricing", func(t *testing.T) {
		var resp PricingResponse
		require.NoError(t, json.Unmarshal([]byte(`{"status":"SUCCESS","pricing":[]}`), &resp))
		assert.Empty(t, resp.Pricing)
	})

	t.Run("dnssec records", func(t *testing.T) {
		var resp GetDnssecRecordsResponse
		require.NoError(t, json.Unmarshal([]byte(`{"status":"SUCCESS","records":[]}`), &resp))
		assert.Empty(t, resp.Records)
	})

	t.Run("auto-renew results", func(t *testing.T) {
		var resp UpdateAutoRenewResponse
		require.NoError(t, json.Unmarshal([]byte(`{"status":"SUCCESS","results":[]}`), &resp))
		assert.Empty(t, resp.Results)
	})

	t.Run("coupons", func(t *testing.T) {
		var coupons Coupons
		require.NoError(t, json.Unmarshal([]byte(`[]`), &coupons))
		assert.Empty(t, coupons)
	})
}

func TestFlexMap_NullDecodesAsEmptyMap(t *testing.T) {
	var resp GetDnssecRecordsResponse
	require.NoError(t, json.Unmarshal([]byte(`{"status":"SUCCESS","records":null}`), &resp))
	assert.Empty(t, resp.Records)
}

func TestFlexMap_ObjectDecodes(t *testing.T) {
	var resp GetDnssecRecordsResponse
	require.NoError(t, json.Unmarshal(
		[]byte(`{"status":"SUCCESS","records":{"64087":{"keyTag":"64087","alg":"13","digestType":"2","digest":"abc"}}}`),
		&resp))

	require.Len(t, resp.Records, 1)
	assert.Equal(t, "64087", resp.Records["64087"].KeyTag)
	assert.Equal(t, DNSSECAlgorithmECDSAP256SHA256, resp.Records["64087"].Alg)
}

// A non-empty array is not an empty map in disguise, so it stays an error rather
// than silently reading as no entries.
func TestFlexMap_NonEmptyArrayIsAnError(t *testing.T) {
	var coupons Coupons
	err := json.Unmarshal([]byte(`[{"code":"X"}]`), &coupons)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected an object or an empty array")
}
