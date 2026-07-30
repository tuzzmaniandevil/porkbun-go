package porkbun

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccount_GetBalance(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/account/balance", "GET", "/account/balance-success.http")

	resp, err := client.Account.GetBalance(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Equal(t, int64(1234), resp.Balance)
	assert.Equal(t, "$12.34", resp.Display)
	assert.Equal(t, "019fa6cf-aa0e-7083-8d51-92119ff2c113", resp.RequestID)
	assert.Equal(t, http.StatusOK, resp.HTTPResponse.StatusCode)
}

func TestAccount_GetBalance_Error(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/account/balance", "GET", "/account/error.http")

	_, err := client.Account.GetBalance(context.Background())

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrCodeInvalidAPIKeys, errResponse.Code)
	assert.Equal(t, NextActionAuthenticate, errResponse.NextAction.Type)
}

func TestAccount_GetApiSettings(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/account/apiSettings", "GET", "/account/apiSettings-success.http")

	resp, err := client.Account.GetAPISettings(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(50000), *resp.Settings.MonthlySpendLimit)
	assert.Equal(t, int64(2000), *resp.Settings.LowBalanceAlert)
	assert.True(t, resp.Settings.AutoTopup)
	assert.Equal(t, int64(1000), *resp.Settings.TopupThreshold)
	assert.Equal(t, int64(5000), *resp.Settings.TopupAmount)
	assert.Equal(t, int64(12000), resp.MonthlySpend)
}

// The API sends null for every unconfigured spend control, which must stay
// distinguishable from a configured limit of zero.
func TestAccount_GetApiSettings_Unset(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/account/apiSettings", "GET", "/account/apiSettings-unset.http")

	resp, err := client.Account.GetAPISettings(context.Background())

	require.NoError(t, err)
	assert.Nil(t, resp.Settings.MonthlySpendLimit)
	assert.Nil(t, resp.Settings.LowBalanceAlert)
	assert.Nil(t, resp.Settings.TopupThreshold)
	assert.Nil(t, resp.Settings.TopupAmount)
	assert.False(t, resp.Settings.AutoTopup)
	assert.Equal(t, int64(0), resp.MonthlySpend)
}

func TestAccount_CreateInvite(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/account/invite", "POST", "/account/invite-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "user@example.com", data["email"])
			assert.Equal(t, "https://example.org/welcome", data["returnUrl"])
		})

	resp, err := client.Account.CreateInvite(context.Background(), &InviteOptions{
		Email:     "user@example.com",
		ReturnURL: "https://example.org/welcome",
	})

	require.NoError(t, err)
	assert.Equal(t, "inv_5f4dcc3b5aa765d61d8327deb882cf99", resp.InviteToken)
	assert.Equal(t, "https://porkbun.com/signup?invite=inv_5f4dcc3b5aa765d61d8327deb882cf99", resp.InviteURL)
	assert.Equal(t, "2026-07-30 04:14:50", resp.Expires)
}

func TestAccount_CreateInvite_NoOptions(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/account/invite", "POST", "/account/invite-success.http",
		func(data map[string]interface{}) {
			assert.NotContains(t, data, "email")
			assert.NotContains(t, data, "returnUrl")
		})

	resp, err := client.Account.CreateInvite(context.Background(), nil)

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

func TestAccount_InviteStatus(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		wantStatus  InviteState
		wantAccount FlexInt64
	}{
		{name: "pending", fixture: "/account/inviteStatus-pending.http", wantStatus: InvitePending},
		{name: "accepted", fixture: "/account/inviteStatus-accepted.http", wantStatus: InviteAccepted, wantAccount: 884213},
		{name: "expired", fixture: "/account/inviteStatus-expired.http", wantStatus: InviteExpired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMockServer(true)
			defer teardownMockServer()

			mux.HandleFunc("/account/inviteStatus", func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "inv_5f4dcc3b5aa765d61d8327deb882cf99", r.URL.Query().Get("token"))
				fixtureHandler(t, "GET", tt.fixture, nil)(w, r)
			})

			resp, err := client.Account.InviteStatus(context.Background(), "inv_5f4dcc3b5aa765d61d8327deb882cf99")

			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.InviteStatus)
			assert.Equal(t, tt.wantAccount, resp.NewAccountID)
		})
	}
}
