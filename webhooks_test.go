package porkbun

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhooks_EventTypes(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/webhook/eventTypes", "GET", "/webhooks/eventTypes-success.http")

	resp, err := client.Webhooks.EventTypes(context.Background())

	require.NoError(t, err)
	assert.Len(t, resp.EventTypes, 7)
	assert.Contains(t, resp.EventTypes, WebhookEventDomainRegistered)
	assert.Contains(t, resp.EventTypes, WebhookEventDNSRecordDeleted)
}

func TestWebhooks_List(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/webhook/list", "GET", "/webhooks/list-success.http")

	resp, err := client.Webhooks.List(context.Background())

	require.NoError(t, err)
	assert.Len(t, resp.Endpoints, 2)

	active := resp.Endpoints[0]
	assert.Equal(t, FlexInt64(7), active.ID)
	assert.Equal(t, "https://example.com/hooks/porkbun", active.URL)
	assert.Equal(t, "whsec_9f1c8b7a6d5e4f3c2b1a0987", active.Secret)
	assert.Equal(t, []WebhookEventType{WebhookEventAll}, active.Events)
	assert.Equal(t, WebhookStatusActive, active.Status)
	assert.Equal(t, FlexInt64(0), active.ConsecutiveFailures)
	assert.Equal(t, "2026-07-27 22:10:00", *active.LastSuccessDate)
	assert.Nil(t, active.LastFailureDate)
	assert.Nil(t, active.LastError)

	// An endpoint auto-disabled after repeated failures.
	disabled := resp.Endpoints[1]
	assert.Equal(t, WebhookStatusDisabled, disabled.Status)
	assert.Equal(t, FlexInt64(20), disabled.ConsecutiveFailures)
	assert.Equal(t, []WebhookEventType{"dns.*"}, disabled.Events)
	assert.Nil(t, disabled.LastSuccessDate)
	assert.Equal(t, "connection timed out", *disabled.LastError)
}

func TestWebhooks_List_Empty(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/webhook/list", "GET", "/webhooks/list-empty.http")

	resp, err := client.Webhooks.List(context.Background())

	require.NoError(t, err)
	assert.Empty(t, resp.Endpoints)
}

func TestWebhooks_Get(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/webhook/get/7", "GET", "/webhooks/get-success.http")

	resp, err := client.Webhooks.Get(context.Background(), 7)

	require.NoError(t, err)
	assert.Equal(t, FlexInt64(7), resp.Endpoint.ID)
	assert.Equal(t, "whsec_9f1c8b7a6d5e4f3c2b1a0987", resp.Endpoint.Secret)
}

func TestWebhooks_Get_NotFound(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/webhook/get/404", "GET", "/webhooks/get-notfound.http")

	_, err := client.Webhooks.Get(context.Background(), 404)

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrorCode("WEBHOOK_NOT_FOUND"), errResponse.Code)
	assert.Equal(t, http.StatusNotFound, errResponse.HTTPResponse.StatusCode)
}

func TestWebhooks_Create(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/webhook/create", "POST", "/webhooks/create-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, "https://example.com/hooks/porkbun", data["url"])
			assert.Equal(t, []interface{}{"domain.renewed", "dns.*"}, data["events"])
		})

	resp, err := client.Webhooks.Create(context.Background(), "https://example.com/hooks/porkbun",
		[]WebhookEventType{WebhookEventDomainRenewed, "dns.*"})

	require.NoError(t, err)
	assert.Equal(t, FlexInt64(7), resp.Endpoint.ID)
	assert.NotEmpty(t, resp.Endpoint.Secret)
}

// Omitting the event list subscribes to everything, so events must not be sent.
func TestWebhooks_Create_AllEvents(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/webhook/create", "POST", "/webhooks/create-success.http",
		func(data map[string]interface{}) {
			assert.NotContains(t, data, "events")
		})

	resp, err := client.Webhooks.Create(context.Background(), "https://example.com/hooks/porkbun", nil)

	require.NoError(t, err)
	assert.Equal(t, []WebhookEventType{WebhookEventAll}, resp.Endpoint.Events)
}

func TestWebhooks_Create_Error(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/webhook/create", "POST", "/webhooks/create-error.http")

	_, err := client.Webhooks.Create(context.Background(), "http://insecure.example.com", nil)

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrorCode("INVALID_WEBHOOK_URL"), errResponse.Code)
	assert.Equal(t, NextActionFixRequest, errResponse.NextAction.Type)
}

func TestWebhooks_Update(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/webhook/update", "POST", "/webhooks/update-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, float64(7), data["id"])
			assert.Equal(t, "https://example.com/hooks/new", data["url"])
			assert.Equal(t, string(WebhookStatusActive), data["status"])
			assert.NotContains(t, data, "events")
		})

	resp, err := client.Webhooks.Update(context.Background(), 7, &UpdateWebhookOptions{
		URL:    String("https://example.com/hooks/new"),
		Status: WebhookStatusActive,
	})

	require.NoError(t, err)
	assert.Equal(t, FlexInt64(7), resp.Endpoint.ID)
}

// Only the supplied fields change, so a nil options struct sends the id alone.
func TestWebhooks_Update_NoOptions(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/webhook/update", "POST", "/webhooks/update-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, float64(7), data["id"])
			assert.NotContains(t, data, "url")
			assert.NotContains(t, data, "status")
			assert.NotContains(t, data, "events")
		})

	_, err := client.Webhooks.Update(context.Background(), 7, nil)

	require.NoError(t, err)
}

func TestWebhooks_RotateSecret(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/webhook/rotateSecret", "POST", "/webhooks/rotateSecret-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, float64(7), data["id"])
		})

	resp, err := client.Webhooks.RotateSecret(context.Background(), 7)

	require.NoError(t, err)
	assert.Equal(t, "whsec_rotated0000111122223333", resp.Endpoint.Secret)
}

func TestWebhooks_Test(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/webhook/test", "POST", "/webhooks/test-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, float64(7), data["id"])
		})

	resp, err := client.Webhooks.Test(context.Background(), 7)

	require.NoError(t, err)
	assert.Equal(t, "018f9c2c-4444-7c41-9b8a-2f1e6d4c5a92", resp.EventID)
	assert.Equal(t, "Test event queued.", resp.Message)
}

func TestWebhooks_Test_Disabled(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/webhook/test", "POST", "/webhooks/test-disabled.http")

	_, err := client.Webhooks.Test(context.Background(), 8)

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrorCode("WEBHOOK_DISABLED"), errResponse.Code)
	assert.Equal(t, NextActionEnableSetting, errResponse.NextAction.Type)
}

func TestWebhooks_Delete(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/webhook/delete", "POST", "/webhooks/delete-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, float64(7), data["id"])
		})

	resp, err := client.Webhooks.Delete(context.Background(), 7)

	require.NoError(t, err)
	assert.Equal(t, "Webhook endpoint deleted.", resp.Message)
}

func TestWebhooks_ListDeliveries(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/webhook/deliveries", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		assert.Equal(t, "7", query.Get("endpointId"))
		assert.Equal(t, string(DeliveryFailed), query.Get("status"))
		assert.Equal(t, "0", query.Get("start"))
		assert.Equal(t, "25", query.Get("limit"))

		fixtureHandler(t, "GET", "/webhooks/deliveries-success.http", nil)(w, r)
	})

	endpointID, start, limit := int64(7), int64(0), int64(25)
	resp, err := client.Webhooks.ListDeliveries(context.Background(), &DeliveryListOptions{
		EndpointID: &endpointID,
		Status:     Ptr(DeliveryFailed),
		Start:      &start,
		Limit:      &limit,
	})

	require.NoError(t, err)
	assert.Equal(t, FlexInt64(2), resp.Total)
	assert.Equal(t, FlexInt64(0), resp.Start)
	assert.Equal(t, FlexInt64(50), resp.Limit)
	assert.Len(t, resp.Deliveries, 2)

	delivered := resp.Deliveries[0]
	assert.Equal(t, FlexInt64(3001), delivered.ID)
	assert.Equal(t, FlexInt64(7), delivered.EndpointID)
	assert.Equal(t, WebhookEventDomainRenewed, delivered.EventType)
	assert.Equal(t, "018f9c2a-7b3e-7c41-9b8a-2f1e6d4c5a90", delivered.EventID)
	assert.Equal(t, DeliveryDelivered, delivered.Status)
	assert.Equal(t, FlexInt64(200), *delivered.HTTPStatus)
	assert.Nil(t, delivered.LastError)
	assert.Equal(t, "2026-07-27 22:10:00", *delivered.DeliveredDate)
	// The bulky payload is only returned by GetDelivery.
	assert.Empty(t, delivered.Payload)

	failed := resp.Deliveries[1]
	assert.Equal(t, DeliveryFailed, failed.Status)
	assert.Equal(t, FlexInt64(6), failed.Attempts)
	assert.Equal(t, FlexInt64(6), failed.MaxAttempts)
	assert.Equal(t, FlexInt64(500), *failed.HTTPStatus)
	assert.Equal(t, "HTTP 500 from endpoint", *failed.LastError)
	assert.Nil(t, failed.DeliveredDate)
}

func TestWebhooks_ListDeliveries_NoOptions(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/webhook/deliveries", func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.RawQuery)

		fixtureHandler(t, "GET", "/webhooks/deliveries-empty.http", nil)(w, r)
	})

	resp, err := client.Webhooks.ListDeliveries(context.Background(), nil)

	require.NoError(t, err)
	assert.Equal(t, FlexInt64(0), resp.Total)
	assert.Empty(t, resp.Deliveries)
}

func TestWebhooks_GetDelivery(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/webhook/delivery/3001", "GET", "/webhooks/delivery-success.http")

	resp, err := client.Webhooks.GetDelivery(context.Background(), 3001)

	require.NoError(t, err)
	assert.Equal(t, FlexInt64(3001), resp.Delivery.ID)
	assert.JSONEq(t,
		`{"event":"domain.renewed","id":"018f9c2a-7b3e-7c41-9b8a-2f1e6d4c5a90","createdAt":"2026-07-27T22:09:58Z","data":{"domain":"example.com","tld":"com","expireDate":"2027-07-27 22:09:58"}}`,
		string(resp.Delivery.Payload))
}

func TestWebhooks_GetDelivery_NotFound(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/webhook/delivery/404", "GET", "/webhooks/delivery-notfound.http")

	_, err := client.Webhooks.GetDelivery(context.Background(), 404)

	testErrorResponse(t, err)

	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)
	assert.Equal(t, ErrorCode("DELIVERY_NOT_FOUND"), errResponse.Code)
}

// A resend clones the delivery but keeps the original event id, so consumers
// that dedupe on it treat the replay as the same event.
func TestWebhooks_Resend(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixtureRequest(t, "/webhook/resend", "POST", "/webhooks/resend-success.http",
		func(data map[string]interface{}) {
			assert.Equal(t, float64(3001), data["id"])
		})

	resp, err := client.Webhooks.Resend(context.Background(), 3001)

	require.NoError(t, err)
	assert.Equal(t, FlexInt64(3003), resp.Delivery.ID)
	assert.Equal(t, DeliveryPending, resp.Delivery.Status)
	assert.Equal(t, FlexInt64(0), resp.Delivery.Attempts)
	assert.Equal(t, "018f9c2a-7b3e-7c41-9b8a-2f1e6d4c5a90", resp.Delivery.EventID)
	assert.Equal(t, "Delivery re-queued.", resp.Message)
}

// setQueryParam skips a nil pointer, so an unset filter must leave no key in the
// query at all, rather than a key carrying the literal "<nil>".
func TestWebhooks_ListDeliveries_OmitsUnsetFilters(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	var rawQuery string
	mux.HandleFunc("/webhook/deliveries", func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS","deliveries":[],"total":0,"start":0,"limit":50}`)
	})

	_, err := client.Webhooks.ListDeliveries(context.Background(), &DeliveryListOptions{
		EndpointID: Ptr(int64(7)),
	})

	require.NoError(t, err)
	assert.Equal(t, "endpointId=7", rawQuery, "only the filter that was set may appear")
}

// Seven webhook methods had no error-path test. They share endpointRequest and
// the GET plumbing, so one table walks them all: a failure the API reports must
// surface as an *ErrorResponse with its code intact, not as a silent success.
func TestWebhooks_ErrorPaths(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	for _, path := range []string{
		"/webhook/eventTypes", "/webhook/list", "/webhook/update", "/webhook/rotateSecret",
		"/webhook/delete", "/webhook/deliveries", "/webhook/resend",
	} {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"status":"ERROR","message":"Webhook endpoint not found.","code":"DOMAIN_NOT_FOUND"}`)
		})
	}

	ctx := context.Background()
	calls := map[string]func() error{
		"EventTypes":     func() error { _, err := client.Webhooks.EventTypes(ctx); return err },
		"List":           func() error { _, err := client.Webhooks.List(ctx); return err },
		"Update":         func() error { _, err := client.Webhooks.Update(ctx, 42, nil); return err },
		"RotateSecret":   func() error { _, err := client.Webhooks.RotateSecret(ctx, 42); return err },
		"Delete":         func() error { _, err := client.Webhooks.Delete(ctx, 42); return err },
		"ListDeliveries": func() error { _, err := client.Webhooks.ListDeliveries(ctx, nil); return err },
		"Resend":         func() error { _, err := client.Webhooks.Resend(ctx, 9001); return err },
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			var apiErr *ErrorResponse
			require.ErrorAs(t, call(), &apiErr)
			assert.Equal(t, ErrCodeDomainNotFound, apiErr.Code)
			assert.Equal(t, StatusError, apiErr.Status)
		})
	}
}
