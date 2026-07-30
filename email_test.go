package porkbun

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmail_SetPassword(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/email/setPassword", "POST", "/email/setPassword-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "user@example.com", data["emailAddress"])
			assert.Equal(t, "correct-horse-battery-staple", data["password"])
		})

	resp, err := client.Email.SetPassword(context.Background(), "user@example.com", "correct-horse-battery-staple")

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Equal(t, "Password updated.", resp.Message)
}

func TestEmail_SetPassword_Error(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/email/setPassword", "POST", "/email/setPassword-error.http")

	_, err := client.Email.SetPassword(context.Background(), "user@example.com", "short")

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrorCode("INVALID_PASSWORD"), errResponse.Code)
	assert.Equal(t, NextActionFixRequest, errResponse.NextAction.Type)
}
