package porkbun

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomains_TransferDomain(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/transfer/example.com", "POST", "/domains/transfer-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "AUTH-CODE-123", data["authCode"])
			assert.Equal(t, float64(999), data["cost"])
			assert.NotContains(t, data, "dryRun")
		})

	resp, err := client.Domains.TransferDomain(context.Background(), "example.com", "AUTH-CODE-123", 999)

	require.NoError(t, err)
	assert.Equal(t, "example.com", resp.Domain)
	assert.Equal(t, FlexInt64(12345680), resp.OrderID)
	assert.Equal(t, FlexInt64(98765), resp.TransferID)
	assert.Equal(t, int64(2829), resp.Balance)
	assert.Contains(t, resp.Message, "5-7 days")
}

func TestDomains_TransferDomain_DryRun(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/transfer/example.com", "POST", "/domains/transfer-dryrun.http",
		func(data map[string]interface{}) {
			assert.Equal(t, true, data["dryRun"])
		})

	resp, err := client.Domains.TransferDomain(context.Background(), "example.com", "AUTH-CODE-123", 999, WithDryRun())

	require.NoError(t, err)
	assert.True(t, resp.DryRun)
	assert.True(t, resp.WouldSucceed)
	assert.Equal(t, OperationTransfer, resp.Operation)
	assert.Equal(t, "unavailable", resp.Available)
	assert.Equal(t, "$9.99", resp.CostDisplay)
	// cost is required on a dry-run preview: it is the amount a live call would
	// charge, so dropping it defeats the purpose of previewing.
	assert.Equal(t, int64(999), resp.Cost)
	assert.Equal(t, FlexInt64(0), resp.TransferID)
}

// The order id arrives as a JSON string on this endpoint, which FlexInt64
// decodes to the same value as a number would.
func TestDomains_GetTransfer(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/getTransfer/tuzzatech.co", "GET", "/domains/getTransfer-success.http")

	resp, err := client.Domains.GetTransfer(context.Background(), "tuzzatech.co")

	require.NoError(t, err)
	assert.Equal(t, "tuzzatech.co", resp.Transfer.Domain)
	assert.Equal(t, TransferDone, resp.Transfer.Status)
	assert.Equal(t, "Transfer completed successfully.", resp.Transfer.StatusDescription)
	assert.Equal(t, "2025-07-09 21:48:57", resp.Transfer.TransferDate)
	assert.Equal(t, FlexInt64(7891021), resp.Transfer.OrderID)
}

func TestDomains_ListTransfers(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/listTransfers", "GET", "/domains/listTransfers-success.http")

	resp, err := client.Domains.ListTransfers(context.Background())

	require.NoError(t, err)
	assert.Len(t, resp.Transfers, 2)
	assert.Equal(t, TransferPendingTransfer, resp.Transfers[0].Status)
	assert.Equal(t, FlexInt64(7891022), resp.Transfers[0].OrderID)
	assert.Equal(t, TransferPendingAuth, resp.Transfers[1].Status)
	assert.Equal(t, "other.com", resp.Transfers[1].Domain)
}

func TestDomains_ListTransfers_Empty(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/listTransfers", "GET", "/domains/listTransfers-empty.http")

	resp, err := client.Domains.ListTransfers(context.Background())

	require.NoError(t, err)
	assert.Empty(t, resp.Transfers)
}
