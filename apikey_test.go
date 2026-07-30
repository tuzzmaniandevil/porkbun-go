package porkbun

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCodeVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

func TestApiKey_Request_PKCE(t *testing.T) {
	// The authorization flow takes no credentials, so the client has none.
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureNoAuth(t, "/apikey/request", "POST", "/apikey/request-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "Acme deploy bot", data["name"])
			assert.Equal(t, "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", data["codeChallenge"])
			assert.Equal(t, "S256", data["codeChallengeMethod"])
		})

	resp, err := client.APIKey.Request(context.Background(), &APIKeyRequestOptions{
		Name:         "Acme deploy bot",
		CodeVerifier: testCodeVerifier,
	})

	require.NoError(t, err)
	assert.Equal(t, DeliveryModePKCE, resp.DeliveryMode)
	// The token is the 64-character lowercase hex string /apikey/retrieve expects.
	assert.Regexp(t, `^[a-f0-9]{64}$`, resp.RequestToken)
	assert.Equal(t, "https://porkbun.com/account/api/authorize/9f1c8b7a", resp.AuthURL)
	assert.Equal(t, "2026-07-28T04:24:50Z", resp.Expiration)

	// The authorization endpoints are the ones that report their limits in
	// headers rather than in the body, so the fixture's headers have to reach the
	// client for these accessors to mean anything.
	limits, ok := resp.RateLimit()
	require.True(t, ok, "the fixture's rate limit headers should reach the client")
	assert.Equal(t, int64(20), limits.Limit)
	assert.Equal(t, int64(19), limits.Remaining)
	assert.Equal(t, int64(3600), limits.Reset)
	assert.Equal(t, "3.9", resp.APIVersion())
}

// Without a code verifier the request must not carry PKCE fields; their absence
// selects the legacy flow, where the secret is only shown in the browser.
func TestApiKey_Request_Legacy(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureNoAuth(t, "/apikey/request", "POST", "/apikey/request-legacy.http",
		func(data map[string]interface{}) {
			assert.NotContains(t, data, "codeChallenge")
			assert.NotContains(t, data, "codeChallengeMethod")
		})

	resp, err := client.APIKey.Request(context.Background(), &APIKeyRequestOptions{Name: "Legacy app"})

	require.NoError(t, err)
	assert.Equal(t, DeliveryModeLegacy, resp.DeliveryMode)
}

func TestApiKey_Request_NoOptions(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureNoAuth(t, "/apikey/request", "POST", "/apikey/request-legacy.http",
		func(data map[string]interface{}) {
			assert.NotContains(t, data, "name")
		})

	resp, err := client.APIKey.Request(context.Background(), nil)

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
}

func TestApiKey_Request_RateLimited(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureNoAuth(t, "/apikey/request", "POST", "/apikey/request-ratelimited.http", nil)

	_, err := client.APIKey.Request(context.Background(), nil)

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrCodeRateLimitExceeded, errResponse.Code)
	assert.Equal(t, int64(1800), errResponse.TTLRemaining)
}

func TestApiKey_Retrieve_Pending(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureNoAuth(t, "/apikey/retrieve", "POST", "/apikey/retrieve-pending.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "abc", data["requestToken"])
			assert.NotContains(t, data, "codeVerifier")
		})

	resp, err := client.APIKey.Retrieve(context.Background(), "abc", "")

	require.NoError(t, err)
	assert.Equal(t, "PENDING", resp.Status)
	assert.Empty(t, resp.APIKey)
	assert.Empty(t, resp.SecretAPIKey)
}

// A PKCE retrieve is the only path that ever returns the secret key.
func TestApiKey_Retrieve_PKCE(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureNoAuth(t, "/apikey/retrieve", "POST", "/apikey/retrieve-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "abc", data["requestToken"])
			assert.Equal(t, testCodeVerifier, data["codeVerifier"])
		})

	resp, err := client.APIKey.Retrieve(context.Background(), "abc", testCodeVerifier)

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Equal(t, "pk1_abcdef0123456789", resp.APIKey)
	assert.Equal(t, "sk1_fedcba9876543210", resp.SecretAPIKey)
}

func TestApiKey_Retrieve_Denied(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureNoAuth(t, "/apikey/retrieve", "POST", "/apikey/retrieve-denied.http", nil)

	_, err := client.APIKey.Retrieve(context.Background(), "abc", testCodeVerifier)

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrorCode("REQUEST_DENIED"), errResponse.Code)
}

func TestPKCECodeChallenge(t *testing.T) {
	// Reference vector from RFC 7636 appendix B.
	assert.Equal(t,
		"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		PKCECodeChallenge(testCodeVerifier))
}

func TestNewPKCECodeVerifier(t *testing.T) {
	first, err := NewPKCECodeVerifier()
	require.NoError(t, err)

	second, err := NewPKCECodeVerifier()
	require.NoError(t, err)

	// 43 unpadded base64url characters, the length the API accepts.
	assert.Len(t, first, 43)
	assert.Regexp(t, `^[A-Za-z0-9\-_]{43}$`, first)
	assert.NotEqual(t, first, second)
}
