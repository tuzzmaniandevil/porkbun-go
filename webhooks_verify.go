package porkbun

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Headers Porkbun sends with every webhook delivery.
const (
	WebhookSignatureHeader = "X-Porkbun-Signature"         // "sha256=" followed by the hex signature
	WebhookTimestampHeader = "X-Porkbun-Webhook-Timestamp" // Unix seconds the request was signed at
	WebhookEventHeader     = "X-Porkbun-Event"             // The event type, e.g. "domain.renewed"
	WebhookIDHeader        = "X-Porkbun-Webhook-Id"        // Event UUIDv7, stable across resends
)

// DefaultWebhookTolerance is how far a delivery's timestamp may drift from the
// current time before VerifyWebhookSignature rejects it as a possible replay.
const DefaultWebhookTolerance = 5 * time.Minute

// Errors returned when a webhook delivery cannot be trusted. Treat every one of
// them as "reject the request", never as "process it anyway".
var (
	ErrWebhookNoSignature      = errors.New("porkbun: webhook is missing its signature header")
	ErrWebhookNoTimestamp      = errors.New("porkbun: webhook is missing its timestamp header")
	ErrWebhookBadTimestamp     = errors.New("porkbun: webhook timestamp is not a unix time")
	ErrWebhookTimestampExpired = errors.New("porkbun: webhook timestamp is outside the allowed tolerance")
	ErrWebhookBadSignature     = errors.New("porkbun: webhook signature does not match")
	ErrWebhookNoSecret         = errors.New("porkbun: webhook signing secret is empty")
)

// WebhookEvent is the envelope Porkbun POSTs to a subscribed endpoint.
type WebhookEvent struct {
	Event     WebhookEventType `json:"event"`     // The event type
	ID        string           `json:"id"`        // UUIDv7, also sent as the X-Porkbun-Webhook-Id header
	CreatedAt time.Time        `json:"createdAt"` // When the event was raised
	Data      RawJSON          `json:"data"`      // Event-specific payload
}

// SignWebhook returns the value Porkbun puts in the X-Porkbun-Signature header
// for a delivery: "sha256=" followed by HMAC-SHA256(secret, "{timestamp}.{body}").
func SignWebhook(secret string, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifyWebhookSignature checks a delivery's signature against the endpoint's
// signing secret, and rejects timestamps that have drifted further than
// tolerance from now.
//
// A zero tolerance applies DefaultWebhookTolerance, so the forgotten argument is
// the safe one. Pass a negative tolerance to disable the replay check, which is
// only reasonable if you deduplicate on the event id instead.
//
// body MUST be the exact bytes received, verified before being parsed: any
// re-encoding changes the signature.
func VerifyWebhookSignature(secret string, body []byte, signature string, timestamp string, tolerance time.Duration) error {
	// An empty secret would verify anything an attacker can HMAC with a known
	// key, so treat an unset secret as a failure rather than a permissive default.
	if secret == "" {
		return ErrWebhookNoSecret
	}

	if signature == "" {
		return ErrWebhookNoSignature
	}

	if timestamp == "" {
		return ErrWebhookNoTimestamp
	}

	signedAt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: %q", ErrWebhookBadTimestamp, timestamp)
	}

	if tolerance == 0 {
		tolerance = DefaultWebhookTolerance
	}

	if tolerance > 0 {
		// Compared in whole seconds, because the header is unix seconds and
		// Duration arithmetic on an absurd timestamp saturates at math.MinInt64,
		// whose negation is itself: an abs() there silently passes the check.
		toleranceSeconds := int64(tolerance / time.Second)
		now := time.Now().Unix()

		if signedAt > now+toleranceSeconds || signedAt < now-toleranceSeconds {
			return fmt.Errorf("%w: signed at %d, now %d", ErrWebhookTimestampExpired, signedAt, now)
		}
	}

	// Constant-time comparison, so a wrong signature leaks nothing by timing.
	expected := SignWebhook(secret, timestamp, body)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrWebhookBadSignature
	}

	return nil
}

// ParseWebhook verifies an incoming delivery against the endpoint's signing
// secret and returns the decoded event. It reads r.Body and replaces it, so the
// caller can still read the raw bytes afterwards.
//
// Deliveries whose timestamp has drifted further than DefaultWebhookTolerance
// are rejected; call VerifyWebhookSignature directly for a different window.
//
// An error means the delivery is not trustworthy: respond with a 4xx and do not
// act on the payload.
//
//	event, err := porkbun.ParseWebhook(r, endpointSecret)
//	if err != nil {
//	    http.Error(w, "invalid webhook", http.StatusBadRequest)
//	    return
//	}
func ParseWebhook(r *http.Request, secret string) (*WebhookEvent, error) {
	// A server request always has a non-nil Body, but a hand-built one need not.
	if r.Body == nil {
		return nil, errors.New("porkbun: webhook request has no body")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("porkbun: reading webhook body: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	err = VerifyWebhookSignature(
		secret,
		body,
		strings.TrimSpace(r.Header.Get(WebhookSignatureHeader)),
		strings.TrimSpace(r.Header.Get(WebhookTimestampHeader)),
		DefaultWebhookTolerance,
	)
	if err != nil {
		return nil, err
	}

	event := &WebhookEvent{}
	if err := json.Unmarshal(body, event); err != nil {
		return nil, fmt.Errorf("porkbun: decoding webhook payload: %w", err)
	}

	return event, nil
}
