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

func TestPricingService_ListPricing_success(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/pricing/get", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/pricing/success.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)

		w.WriteHeader(httpResponse.StatusCode)
		_, _ = io.Copy(w, httpResponse.Body)
	})

	resp, err := client.Pricing.ListPricing(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.NotEmpty(t, resp.Pricing["com"].Registration, "registration price must decode")
	assert.NotEmpty(t, resp.Pricing["com"].Renewal)
	assert.NotEmpty(t, resp.Pricing["com"].Transfer)
	assert.NotEmpty(t, resp.Pricing)
	assert.NotNil(t, resp.Pricing["com.mx"])
	assert.NotNil(t, resp.Pricing["com.mx"].Coupons)
	assert.NotNil(t, resp.Pricing["com.mx"].Coupons["registration"])
	assert.Equal(t, "AWESOMENESS", resp.Pricing["com.mx"].Coupons["registration"].Code)
}

func TestPricingService_ListPricing_InvalidCouponType(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/pricing/get", func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, "/pricing/success-invalidcoupon.http")
		assert.NotNil(t, httpResponse)

		testMethod(t, r, "POST")
		testHeaders(t, r)

		w.WriteHeader(httpResponse.StatusCode)
		_, _ = io.Copy(w, httpResponse.Body)
	})

	_, err := client.Pricing.ListPricing(context.Background())

	assert.Error(t, err)
}

func TestPricingService_ListPricing_EmptyResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/pricing/get", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS","pricing":{}}`)
	})

	resp, err := client.Pricing.ListPricing(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Empty(t, resp.Pricing)
}

func TestPricingService_ListPricing_MalformedResponse(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/pricing/get", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS","pricing":`)
	})

	_, err := client.Pricing.ListPricing(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
}

// Pricing is public: the client must send no credentials even when it has them,
// which handleFixtureNoAuth asserts for both the body and the headers.
func TestPricing_ListPricing_TldFilter(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureNoAuth(t, "/pricing/get", "POST", "/pricing/success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, []interface{}{"com", "io"}, data["tlds"])
		})

	resp, err := client.Pricing.ListPricing(context.Background(), "com", "io")

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Pricing)
}
