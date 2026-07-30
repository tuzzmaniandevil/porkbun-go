package porkbun

import (
	"context"
	"net/url"
)

// WebhookService provides methods to interact with the webhook API.
type WebhookService struct {
	client *Client // Client used to communicate with the API
}

// WebhookEndpoint represents a registered webhook endpoint.
type WebhookEndpoint struct {
	ID                  FlexInt64          `json:"id"`                  // Numeric endpoint id
	URL                 string             `json:"url"`                 // Destination HTTPS URL
	Secret              string             `json:"secret"`              // HMAC-SHA256 signing secret used to verify X-Porkbun-Signature
	Events              []WebhookEventType `json:"events"`              // Subscribed event types, or ["*"] for all
	Status              WebhookStatus      `json:"status"`              // WebhookStatusActive or WebhookStatusDisabled
	ConsecutiveFailures FlexInt64          `json:"consecutiveFailures"` // Consecutive failed deliveries
	LastSuccessDate     *string            `json:"lastSuccessDate"`     // Timestamp of the last successful delivery
	LastFailureDate     *string            `json:"lastFailureDate"`     // Timestamp of the last failed delivery
	LastError           *string            `json:"lastError"`           // Last delivery error
	CreateDate          string             `json:"createDate"`          // When the endpoint was created
}

// WebhookDelivery represents a single webhook delivery attempt.
type WebhookDelivery struct {
	ID            FlexInt64        `json:"id"`                // Delivery id
	EndpointID    FlexInt64        `json:"endpointId"`        // Endpoint this delivery targets
	EventType     WebhookEventType `json:"eventType"`         // The event type delivered
	EventID       string           `json:"eventId"`           // UUIDv7 of the event, stable across resends
	Status        DeliveryStatus   `json:"status"`            // Delivery state, e.g. DeliveryDelivered
	Attempts      FlexInt64        `json:"attempts"`          // Delivery attempts made so far
	MaxAttempts   FlexInt64        `json:"maxAttempts"`       // Attempt cap before the delivery is marked FAILED
	HTTPStatus    *FlexInt64       `json:"httpStatus"`        // HTTP status from the last attempt
	LastError     *string          `json:"lastError"`         // Last delivery error
	NextAttemptAt *string          `json:"nextAttemptAt"`     // When the next attempt is scheduled
	CreateDate    string           `json:"createDate"`        // When the delivery was queued
	DeliveredDate *string          `json:"deliveredDate"`     // When it was successfully delivered
	Payload       RawJSON          `json:"payload,omitempty"` // The full event envelope, only on GetDelivery
}

// WebhookEventTypesResponse represents the response structure for the event type catalog.
type WebhookEventTypesResponse struct {
	BaseResponse
	EventTypes []WebhookEventType `json:"eventTypes"` // Event types an endpoint can subscribe to
}

// WebhookListResponse represents the response structure for listing endpoints.
type WebhookListResponse struct {
	BaseResponse
	Endpoints []WebhookEndpoint `json:"endpoints"` // The registered endpoints
}

// WebhookEndpointResponse represents the response structure for a single endpoint.
type WebhookEndpointResponse struct {
	BaseResponse
	Endpoint WebhookEndpoint `json:"endpoint"` // The endpoint
}

// createWebhookRequest represents the request structure for creating an endpoint.
type createWebhookRequest struct {
	baseRequest
	URL    string             `json:"url"`              // HTTPS URL to deliver events to
	Events []WebhookEventType `json:"events,omitempty"` // Event types to subscribe to. Omit, or pass WebhookEventAll, for all
}

// updateWebhookRequest represents the request structure for updating an endpoint.
// Only the supplied fields change.
type updateWebhookRequest struct {
	baseRequest
	UpdateWebhookOptions
	ID int64 `json:"id"` // Endpoint id to update
}

// webhookIDRequest represents the request structure for endpoint operations that
// take only an id.
type webhookIDRequest struct {
	baseRequest
	ID int64 `json:"id"` // Endpoint or delivery id
}

// WebhookTestResponse represents the response structure for sending a test event.
type WebhookTestResponse struct {
	BaseResponse
	EventID string `json:"eventId"` // UUID of the queued webhook.test event
}

// WebhookDeleteResponse represents the response structure for deleting an endpoint.
type WebhookDeleteResponse struct {
	BaseResponse
}

// WebhookDeliveryListResponse represents the response structure for listing deliveries.
type WebhookDeliveryListResponse struct {
	BaseResponse
	Deliveries []WebhookDelivery `json:"deliveries"` // Newest first, without the bulky payload field
	Total      FlexInt64         `json:"total"`      // Total matching deliveries
	Start      FlexInt64         `json:"start"`      // Offset of this page
	Limit      FlexInt64         `json:"limit"`      // Page size used
}

// WebhookDeliveryResponse represents the response structure for a single delivery.
type WebhookDeliveryResponse struct {
	BaseResponse
	Delivery WebhookDelivery `json:"delivery"` // The delivery, including its full payload
}

// UpdateWebhookOptions provides the fields that can be changed on an endpoint.
// Leave a field at its zero value to keep the current one.
type UpdateWebhookOptions struct {
	URL    *string            `json:"url,omitempty"`    // New HTTPS URL
	Events []WebhookEventType `json:"events,omitempty"` // Replacement event subscription list
	Status WebhookStatus      `json:"status,omitempty"` // WebhookStatusActive to resume and clear the failure counter, or WebhookStatusDisabled to pause
}

// DeliveryListOptions provides options for filtering webhook deliveries.
type DeliveryListOptions struct {
	EndpointID *int64          // Only deliveries for this endpoint
	Status     *DeliveryStatus // Filter by delivery status, e.g. DeliveryFailed
	Start      *int64          // Pagination offset, default 0
	Limit      *int64          // Page size, 1-200, default 50
}

// EventTypes returns the catalog of event types an endpoint can subscribe to.
func (s *WebhookService) EventTypes(ctx context.Context) (*WebhookEventTypesResponse, error) {
	response := &WebhookEventTypesResponse{}

	_, err := s.client.get(ctx, apiPath("webhook", "eventTypes"), response)
	return response, err
}

// List returns all webhook endpoints registered on the account, including each
// endpoint's signing secret and delivery health.
func (s *WebhookService) List(ctx context.Context) (*WebhookListResponse, error) {
	response := &WebhookListResponse{}

	_, err := s.client.get(ctx, apiPath("webhook", "list"), response)
	return response, err
}

// Get returns a single webhook endpoint by id.
func (s *WebhookService) Get(ctx context.Context, id int64) (*WebhookEndpointResponse, error) {
	response := &WebhookEndpointResponse{}

	_, err := s.client.get(ctx, apiPath("webhook", "get", id), response)
	return response, err
}

// Create registers an HTTPS endpoint to receive signed event payloads. The
// response includes the generated signing secret, which is the HMAC key used to
// verify the X-Porkbun-Signature header. Pass no events, or ["*"], for all events.
func (s *WebhookService) Create(ctx context.Context, endpointURL string, events []WebhookEventType, opts ...RequestOption) (*WebhookEndpointResponse, error) {
	request := &createWebhookRequest{
		URL:    endpointURL,
		Events: events,
	}

	return s.endpointRequest(ctx, apiPath("webhook", "create"), request, opts...)
}

// Update changes an endpoint's URL, event subscriptions and/or status.
func (s *WebhookService) Update(ctx context.Context, id int64, options *UpdateWebhookOptions, opts ...RequestOption) (*WebhookEndpointResponse, error) {
	request := &updateWebhookRequest{ID: id}

	if options != nil {
		request.UpdateWebhookOptions = *options
	}

	return s.endpointRequest(ctx, apiPath("webhook", "update"), request, opts...)
}

// RotateSecret generates a new signing secret for an endpoint and returns the
// endpoint with that secret. Deliveries are signed with it immediately.
func (s *WebhookService) RotateSecret(ctx context.Context, id int64, opts ...RequestOption) (*WebhookEndpointResponse, error) {
	return s.endpointRequest(ctx, apiPath("webhook", "rotateSecret"), &webhookIDRequest{ID: id}, opts...)
}

// Test enqueues a webhook.test event to the endpoint, which must be ACTIVE.
// Delivery is asynchronous, usually within a minute.
func (s *WebhookService) Test(ctx context.Context, id int64, opts ...RequestOption) (*WebhookTestResponse, error) {
	request := &webhookIDRequest{ID: id}
	response := &WebhookTestResponse{}

	_, err := s.client.post(ctx, apiPath("webhook", "test"), request, response, opts...)
	return response, err
}

// Delete removes a webhook endpoint. Deliveries stop immediately.
func (s *WebhookService) Delete(ctx context.Context, id int64, opts ...RequestOption) (*WebhookDeleteResponse, error) {
	request := &webhookIDRequest{ID: id}
	response := &WebhookDeleteResponse{}

	_, err := s.client.post(ctx, apiPath("webhook", "delete"), request, response, opts...)
	return response, err
}

// ListDeliveries returns recent delivery attempts across the account, newest
// first. The bulky payload field is omitted, use GetDelivery for it.
func (s *WebhookService) ListDeliveries(ctx context.Context, options *DeliveryListOptions) (*WebhookDeliveryListResponse, error) {
	params := url.Values{}
	if options != nil {
		setQueryParam(params, "endpointId", options.EndpointID)
		setQueryParam(params, "status", options.Status)
		setQueryParam(params, "start", options.Start)
		setQueryParam(params, "limit", options.Limit)
	}

	response := &WebhookDeliveryListResponse{}
	_, err := s.client.get(ctx, buildQuery(apiPath("webhook", "deliveries"), params), response)
	return response, err
}

// GetDelivery returns a single delivery, including the full event payload.
func (s *WebhookService) GetDelivery(ctx context.Context, id int64) (*WebhookDeliveryResponse, error) {
	response := &WebhookDeliveryResponse{}

	_, err := s.client.get(ctx, apiPath("webhook", "delivery", id), response)
	return response, err
}

// Resend re-queues a past delivery, reusing the original event id so consumers
// that dedupe on X-Porkbun-Webhook-ID treat it as the same event.
func (s *WebhookService) Resend(ctx context.Context, deliveryID int64, opts ...RequestOption) (*WebhookDeliveryResponse, error) {
	request := &webhookIDRequest{ID: deliveryID}
	response := &WebhookDeliveryResponse{}

	_, err := s.client.post(ctx, apiPath("webhook", "resend"), request, response, opts...)
	return response, err
}

// endpointRequest posts a request that returns a single endpoint.
func (s *WebhookService) endpointRequest(ctx context.Context, path string, request any, opts ...RequestOption) (*WebhookEndpointResponse, error) {
	response := &WebhookEndpointResponse{}

	_, err := s.client.post(ctx, path, request, response, opts...)
	return response, err
}
