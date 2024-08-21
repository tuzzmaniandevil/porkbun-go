package porkbun

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPricingService_ListPricing_success(t *testing.T) {
	setupMockServer(false)
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

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.NotEmpty(t, resp.Pricing)
	assert.NotNil(t, resp.Pricing["com.mx"])
	assert.NotNil(t, resp.Pricing["com.mx"].Coupons)
	assert.NotNil(t, resp.Pricing["com.mx"].Coupons["registration"])
	assert.Equal(t, "AWESOMENESS", resp.Pricing["com.mx"].Coupons["registration"].Code)
}

func TestPricingService_ListPricing_InvalidCouponType(t *testing.T) {
	setupMockServer(false)
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
	setupMockServer(false)
	defer teardownMockServer()

	mux.HandleFunc("/pricing/get", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"SUCCESS","pricing":{}}`)
	})

	resp, err := client.Pricing.ListPricing(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp.Status)
	assert.Empty(t, resp.Pricing)
}

func TestPricingService_ListPricing_MalformedResponse(t *testing.T) {
	setupMockServer(false)
	defer teardownMockServer()

	mux.HandleFunc("/pricing/get", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"SUCCESS","pricing":`)
	})

	_, err := client.Pricing.ListPricing(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected end of JSON input")
}
