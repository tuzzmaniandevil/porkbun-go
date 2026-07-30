package porkbun

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomains_CheckDomain(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/checkDomain/example.com", "POST", "/domains/checkDomain-success.http")

	resp, err := client.Domains.CheckDomain(context.Background(), "example.com")

	require.NoError(t, err)
	assert.Equal(t, Yes, resp.Response.Avail)
	assert.Equal(t, "registration", resp.Response.Type)
	assert.Equal(t, "9.73", resp.Response.Price)
	assert.Equal(t, Yes, resp.Response.FirstYearPromo)
	assert.Equal(t, "11.08", resp.Response.RegularPrice)
	assert.Equal(t, No, resp.Response.Premium)
	assert.Equal(t, int64(1), resp.Response.MinDuration)
	assert.Equal(t, "11.08", resp.Response.Additional.Renewal.Price)
	assert.Equal(t, "transfer", resp.Response.Additional.Transfer.Type)
	assert.Equal(t, int64(10), resp.Limits.TTL)
	assert.Equal(t, int64(1), resp.Limits.Used)
	assert.NotEmpty(t, resp.Limits.NaturalLanguage, "the human-readable summary must decode")
	assert.Equal(t, int64(10), resp.TTLRemaining)
}

func TestDomains_CheckDomain_Unavailable(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/checkDomain/tuzzatech.co", "POST", "/domains/checkDomain-unavailable.http")

	resp, err := client.Domains.CheckDomain(context.Background(), "tuzzatech.co")

	require.NoError(t, err)
	assert.Equal(t, No, resp.Response.Avail)
	assert.Equal(t, "31.20", resp.Response.Additional.Renewal.Price)
}

func TestDomains_CreateDomain(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/create/example.com", "POST", "/domains/create-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, float64(973), data["cost"])
			assert.Equal(t, "yes", data["agreeToTerms"])
			assert.NotContains(t, data, "whoisPrivacy")
			assert.NotContains(t, data, "dryRun")
		})

	resp, err := client.Domains.CreateDomain(context.Background(), "example.com", &CreateDomainOptions{
		Cost:         973,
		AgreeToTerms: "yes",
	})

	require.NoError(t, err)
	assert.Equal(t, "example.com", resp.Domain)
	assert.Equal(t, int64(973), resp.Cost)
	assert.Equal(t, FlexInt64(12345678), resp.OrderID)
	assert.Equal(t, int64(4027), resp.Balance)
	assert.Equal(t, int64(1), resp.Limits.Attempts.Used)
	assert.Equal(t, int64(50), resp.Limits.Success.Limit)
	assert.False(t, resp.DryRun)
}

func TestDomains_CreateDomain_WhoisPrivacy(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/create/example.com", "POST", "/domains/create-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, false, data["whoisPrivacy"])
		})

	whoisPrivacy := false
	_, err := client.Domains.CreateDomain(context.Background(), "example.com", &CreateDomainOptions{
		Cost:         973,
		AgreeToTerms: "yes",
		WhoisPrivacy: &whoisPrivacy,
	})

	require.NoError(t, err)
}

func TestDomains_CreateDomain_DryRun(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/create/example.com", "POST", "/domains/create-dryrun.http",
		func(data map[string]interface{}) {
			assert.Equal(t, true, data["dryRun"])
		})

	resp, err := client.Domains.CreateDomain(context.Background(), "example.com", &CreateDomainOptions{
		Cost:         973,
		AgreeToTerms: "yes",
	}, WithDryRun())

	require.NoError(t, err)
	assert.True(t, resp.DryRun)
	assert.True(t, resp.WouldSucceed)
	assert.Equal(t, OperationRegistration, resp.Operation)
	assert.Equal(t, "com", resp.TLD)
	assert.Equal(t, "available", resp.Available)
	assert.False(t, resp.Premium)
	assert.Equal(t, int64(1), resp.Duration)
	assert.Equal(t, "$9.73", resp.CostDisplay)
	assert.True(t, resp.SufficientFunds)
	require.NotNil(t, resp.MonthlySpendLimit, "the fixture configures a cap")
	assert.Equal(t, int64(50000), *resp.MonthlySpendLimit)
	require.NotNil(t, resp.MonthlySpendSoFar)
	assert.Equal(t, int64(12000), *resp.MonthlySpendSoFar)
	require.NotNil(t, resp.WithinMonthlySpendLimit)
	assert.True(t, *resp.WithinMonthlySpendLimit)
	// A preview creates no order.
	assert.Equal(t, FlexInt64(0), resp.OrderID)
}

func TestDomains_CreateDomain_InsufficientFunds(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/create/example.com", "POST", "/domains/create-insufficientfunds.http")

	_, err := client.Domains.CreateDomain(context.Background(), "example.com", &CreateDomainOptions{
		Cost:         973,
		AgreeToTerms: "yes",
	})

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrCodeInsufficientFunds, errResponse.Code)
	assert.Equal(t, NextActionAddFunds, errResponse.NextAction.Type)
	assert.NotEmpty(t, errResponse.NextAction.Hint, "the remediation hint must decode")
}

func TestDomains_RenewDomain(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/renew/example.com", "POST", "/domains/renew-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, float64(1099), data["cost"])
			assert.NotContains(t, data, "dryRun")
		})

	resp, err := client.Domains.RenewDomain(context.Background(), "example.com", 1099)

	require.NoError(t, err)
	assert.Equal(t, "example.com", resp.Domain)
	assert.Equal(t, "2027-04-21", resp.ExpirationDate)
	assert.Equal(t, int64(1099), resp.Cost)
	assert.Equal(t, FlexInt64(12345679), resp.OrderID)
	assert.Equal(t, int64(3928), resp.Balance)
	assert.Equal(t, int64(86400), resp.TTLRemaining)
}

func TestDomains_RenewDomain_DryRun(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/domain/renew/example.com", "POST", "/domains/renew-dryrun.http",
		func(data map[string]interface{}) {
			assert.Equal(t, true, data["dryRun"])
		})

	resp, err := client.Domains.RenewDomain(context.Background(), "example.com", 1099, WithDryRun())

	require.NoError(t, err)
	assert.True(t, resp.DryRun)
	assert.True(t, resp.WouldSucceed)
	assert.Equal(t, OperationRenewal, resp.Operation)
	assert.Equal(t, "$10.99", resp.CostDisplay)
	assert.Equal(t, FlexInt64(0), resp.OrderID)
}

func TestDomains_RenewDomain_Error(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/renew/example.com", "POST", "/domains/renew-error.http")

	_, err := client.Domains.RenewDomain(context.Background(), "example.com", 1099)

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrorCode("RENEWAL_TOO_SOON"), errResponse.Code)
	assert.Equal(t, NextActionWaitAndRetry, errResponse.NextAction.Type)
}

func TestDomains_GetRegistrationRequirements(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/getRegistrationRequirements/com", "GET", "/domains/getRegistrationRequirements-com.http")

	resp, err := client.Domains.GetRegistrationRequirements(context.Background(), "com")

	require.NoError(t, err)
	assert.Equal(t, "com", resp.TLD)
	assert.True(t, resp.APIRegisterable)
	assert.Equal(t, int64(1), resp.RegistrationDurationYears)
	require.NotNil(t, resp.MaxRegistrationYears)
	assert.Equal(t, int64(10), *resp.MaxRegistrationYears)
	assert.True(t, resp.WhoisPrivacySupported)
	assert.False(t, resp.RequiresValidatedAddress)
	assert.False(t, resp.RegistrantOnly)
	assert.JSONEq(t,
		`{"type":"object","required":["cost","agreeToTerms"],"properties":{"cost":{"type":"integer"},"agreeToTerms":{"enum":["yes","1"]}}}`,
		string(resp.RequestSchema))
	assert.Empty(t, resp.RegistryRequirements)
}

// A TLD with registry eligibility rules is not API-registerable and returns a
// second schema describing the fields the registry demands.
func TestDomains_GetRegistrationRequirements_NotRegisterable(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/getRegistrationRequirements/us", "GET", "/domains/getRegistrationRequirements-us.http")

	resp, err := client.Domains.GetRegistrationRequirements(context.Background(), "us")

	require.NoError(t, err)
	assert.False(t, resp.APIRegisterable)
	assert.True(t, resp.RequiresValidatedAddress)
	assert.True(t, resp.RegistrantOnly)
	assert.Contains(t, resp.NotAPIRegisterableReason, "registry eligibility requirements")
	assert.JSONEq(t,
		`{"type":"object","required":["nexusCategory"],"properties":{"nexusCategory":{"enum":["C11","C12","C21"]}}}`,
		string(resp.RegistryRequirements))
}

func TestDomains_GetRegistrationRequirements_Error(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/domain/getRegistrationRequirements/nope", "GET", "/domains/getRegistrationRequirements-error.http")

	_, err := client.Domains.GetRegistrationRequirements(context.Background(), "nope")

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrorCode("INVALID_TLD"), errResponse.Code)
}
