package porkbun

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarketplace_ListDomains(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/marketplace/getAll", "POST", "/marketplace/getAll-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, float64(0), data["start"])
			assert.Equal(t, float64(1000), data["limit"])
			assert.NotContains(t, data, "query")
		})

	start, limit := int64(0), int64(1000)
	resp, err := client.Marketplace.ListDomains(context.Background(), &MarketplaceListOptions{
		Start: &start,
		Limit: &limit,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Count)
	assert.False(t, resp.Filtered)
	assert.Len(t, resp.Domains, 2)
	assert.Equal(t, "ai.token", resp.Domains[0].Domain)
	assert.Equal(t, "token", resp.Domains[0].TLD)
	assert.Equal(t, int64(2), resp.Domains[0].SLDLength)
	assert.Equal(t, 8888.0, resp.Domains[0].Price)
	assert.Equal(t, "2022-04-22 02:47:43", resp.Domains[0].CreateDate)
}

// Supplying any filter switches the API into filtered mode.
func TestMarketplace_ListDomains_Filtered(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/marketplace/getAll", "POST", "/marketplace/getAll-filtered.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "ai -test", data["query"])
			assert.Equal(t, []interface{}{"com", "io"}, data["tlds"])
			assert.Equal(t, float64(2), data["sldLengthMin"])
			assert.Equal(t, float64(3), data["sldLengthMax"])
			assert.Equal(t, "price", data["sortName"])
			assert.Equal(t, "desc", data["sortDirection"])
		})

	min, max := int64(2), int64(3)
	resp, err := client.Marketplace.ListDomains(context.Background(), &MarketplaceListOptions{
		Query:         String("ai -test"),
		TLDs:          []string{"com", "io"},
		SLDLengthMin:  &min,
		SLDLengthMax:  &max,
		SortName:      MarketplaceSortByPrice,
		SortDirection: SortDescending,
	})

	require.NoError(t, err)
	assert.True(t, resp.Filtered)
	assert.Equal(t, int64(1), resp.Count)
	// Prices are fractional dollars, not cents.
	assert.Equal(t, 1500.5, resp.Domains[0].Price)
}

func TestMarketplace_ListDomains_NoOptions(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/marketplace/getAll", "POST", "/marketplace/getAll-success.http",
		func(data map[string]interface{}) {
			assert.NotContains(t, data, "start")
			assert.NotContains(t, data, "limit")
			assert.NotContains(t, data, "sortName")
		})

	resp, err := client.Marketplace.ListDomains(context.Background(), nil)

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

func TestMarketplace_ListDomains_Error(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/marketplace/getAll", "POST", "/marketplace/getAll-error.http")

	_, err := client.Marketplace.ListDomains(context.Background(), nil)

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrCodeInvalidAPIKeys, errResponse.Code)
}
