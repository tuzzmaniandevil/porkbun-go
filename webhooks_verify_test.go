package porkbun

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testWebhookSecret = "whsec_9f1c8b7a6d5e4f3c2b1a0987"
	testWebhookBody   = `{"event":"domain.renewed","id":"018f9c2a-7b3e-7c41-9b8a-2f1e6d4c5a90","createdAt":"2026-07-27T22:09:58Z","data":{"domain":"example.com","tld":"com","expireDate":"2027-07-27 22:09:58"}}`
)

// signedNow returns the headers Porkbun would send for body right now.
func signedNow(t *testing.T, body string) (signature string, timestamp string) {
	t.Helper()

	timestamp = strconv.FormatInt(time.Now().Unix(), 10)
	return SignWebhook(testWebhookSecret, timestamp, []byte(body)), timestamp
}

// The signature covers "{timestamp}.{rawBody}", so it changes with either.
func TestSignWebhook(t *testing.T) {
	got := SignWebhook("secret", "1750184400", []byte(`{"event":"domain.renewed"}`))

	assert.True(t, strings.HasPrefix(got, "sha256="))
	assert.Len(t, got, len("sha256=")+64) // hex-encoded SHA-256
	// Stable for the same inputs, and bound to the timestamp.
	assert.Equal(t, got, SignWebhook("secret", "1750184400", []byte(`{"event":"domain.renewed"}`)))
	assert.NotEqual(t, got, SignWebhook("secret", "1750184401", []byte(`{"event":"domain.renewed"}`)))
}

func TestVerifyWebhookSignature(t *testing.T) {
	signature, timestamp := signedNow(t, testWebhookBody)

	err := VerifyWebhookSignature(testWebhookSecret, []byte(testWebhookBody), signature, timestamp, DefaultWebhookTolerance)

	assert.NoError(t, err)
}

func TestVerifyWebhookSignature_Rejects(t *testing.T) {
	signature, timestamp := signedNow(t, testWebhookBody)
	old := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
	future := strconv.FormatInt(time.Now().Add(10*time.Minute).Unix(), 10)

	tests := []struct {
		name      string
		secret    string
		body      string
		signature string
		timestamp string
		tolerance time.Duration
		wantErr   error
	}{
		{
			name: "wrong secret", secret: "whsec_someoneelse", body: testWebhookBody,
			signature: signature, timestamp: timestamp, tolerance: DefaultWebhookTolerance,
			wantErr: ErrWebhookBadSignature,
		},
		{
			name: "tampered body", secret: testWebhookSecret, body: strings.Replace(testWebhookBody, "example.com", "evil.com", 1),
			signature: signature, timestamp: timestamp, tolerance: DefaultWebhookTolerance,
			wantErr: ErrWebhookBadSignature,
		},
		{
			name: "signature for a different timestamp", secret: testWebhookSecret, body: testWebhookBody,
			signature: SignWebhook(testWebhookSecret, "1", []byte(testWebhookBody)), timestamp: timestamp,
			tolerance: DefaultWebhookTolerance, wantErr: ErrWebhookBadSignature,
		},
		{
			name: "missing signature", secret: testWebhookSecret, body: testWebhookBody,
			signature: "", timestamp: timestamp, tolerance: DefaultWebhookTolerance,
			wantErr: ErrWebhookNoSignature,
		},
		{
			name: "missing timestamp", secret: testWebhookSecret, body: testWebhookBody,
			signature: signature, timestamp: "", tolerance: DefaultWebhookTolerance,
			wantErr: ErrWebhookNoTimestamp,
		},
		{
			name: "non-numeric timestamp", secret: testWebhookSecret, body: testWebhookBody,
			signature: signature, timestamp: "yesterday", tolerance: DefaultWebhookTolerance,
			wantErr: ErrWebhookBadTimestamp,
		},
		{
			name: "replay of an old delivery", secret: testWebhookSecret, body: testWebhookBody,
			signature: SignWebhook(testWebhookSecret, old, []byte(testWebhookBody)), timestamp: old,
			tolerance: DefaultWebhookTolerance, wantErr: ErrWebhookTimestampExpired,
		},
		{
			name: "timestamp too far ahead", secret: testWebhookSecret, body: testWebhookBody,
			signature: SignWebhook(testWebhookSecret, future, []byte(testWebhookBody)), timestamp: future,
			tolerance: DefaultWebhookTolerance, wantErr: ErrWebhookTimestampExpired,
		},
		{
			// A prefixless signature must not be accepted.
			name: "signature without the sha256 prefix", secret: testWebhookSecret, body: testWebhookBody,
			signature: strings.TrimPrefix(signature, "sha256="), timestamp: timestamp,
			tolerance: DefaultWebhookTolerance, wantErr: ErrWebhookBadSignature,
		},
		{
			// An unset secret, from a missing environment variable, would
			// otherwise verify any body an attacker can HMAC with an empty key.
			name: "empty secret", secret: "", body: testWebhookBody,
			signature: SignWebhook("", timestamp, []byte(testWebhookBody)), timestamp: timestamp,
			tolerance: DefaultWebhookTolerance, wantErr: ErrWebhookNoSecret,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyWebhookSignature(tt.secret, []byte(tt.body), tt.signature, tt.timestamp, tt.tolerance)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// A negative tolerance disables the replay window, for consumers that dedupe on
// the event id instead. Zero applies the default, so the forgotten argument is
// the safe one rather than the permissive one.
func TestVerifyWebhookSignature_Tolerance(t *testing.T) {
	old := strconv.FormatInt(time.Now().Add(-72*time.Hour).Unix(), 10)
	signature := SignWebhook(testWebhookSecret, old, []byte(testWebhookBody))

	verify := func(tolerance time.Duration) error {
		return VerifyWebhookSignature(testWebhookSecret, []byte(testWebhookBody), signature, old, tolerance)
	}

	assert.NoError(t, verify(-1), "a negative tolerance disables the replay check")
	assert.ErrorIs(t, verify(0), ErrWebhookTimestampExpired, "zero must apply the default, not disable the check")
	assert.ErrorIs(t, verify(DefaultWebhookTolerance), ErrWebhookTimestampExpired)
	assert.NoError(t, verify(96*time.Hour), "a window wide enough to contain it")
}

// A timestamp far enough in the future made Duration arithmetic saturate at
// math.MinInt64, whose negation is itself, so an abs()-based drift check passed
// it. Only Porkbun can sign one, but the guard must not be dead for that class.
func TestVerifyWebhookSignature_RejectsAbsurdTimestamps(t *testing.T) {
	for _, ts := range []string{
		"9223372036854775807",  // math.MaxInt64
		"9223371974719179007",  // saturates time.Since to math.MinInt64
		"1152921504606846976",  // ~2^60 seconds from the epoch
		"14399765950",          // ~year 2426
		"-9223372036854775808", // math.MinInt64
		"-1",
		"0",
	} {
		t.Run(ts, func(t *testing.T) {
			signature := SignWebhook(testWebhookSecret, ts, []byte(testWebhookBody))

			err := VerifyWebhookSignature(testWebhookSecret, []byte(testWebhookBody), signature, ts, DefaultWebhookTolerance)

			assert.ErrorIs(t, err, ErrWebhookTimestampExpired,
				"a correctly signed but absurd timestamp must still be rejected")
		})
	}
}

// A future timestamp is clock skew, not a replay, and the message must say which.
func TestVerifyWebhookSignature_ReportsTheSignedTime(t *testing.T) {
	future := strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)
	signature := SignWebhook(testWebhookSecret, future, []byte(testWebhookBody))

	err := VerifyWebhookSignature(testWebhookSecret, []byte(testWebhookBody), signature, future, DefaultWebhookTolerance)

	require.ErrorIs(t, err, ErrWebhookTimestampExpired)
	assert.Contains(t, err.Error(), "signed at "+future, "the error must not claim a future delivery was signed N ago")
}

// A hand-built request, from a fuzz harness or middleware that swapped the body,
// must not panic inside the SDK.
func TestParseWebhook_NilBody(t *testing.T) {
	request, err := http.NewRequest(http.MethodPost, "https://example.org/hooks", nil)
	require.NoError(t, err)
	request.Body = nil

	event, err := ParseWebhook(request, testWebhookSecret)

	assert.Nil(t, event)
	assert.ErrorContains(t, err, "no body")
}

// Porkbun sends the headers unpadded, but a proxy may not, and a signature that
// verifies must not be rejected over surrounding whitespace.
func TestParseWebhook_TrimsHeaderWhitespace(t *testing.T) {
	request := deliveryRequest(t, testWebhookBody, true)
	request.Header.Set(WebhookSignatureHeader, " "+request.Header.Get(WebhookSignatureHeader)+" ")
	request.Header.Set(WebhookTimestampHeader, "  "+request.Header.Get(WebhookTimestampHeader)+"\t")

	event, err := ParseWebhook(request, testWebhookSecret)

	require.NoError(t, err)
	require.NotNil(t, event)
}

// deliveryRequest builds the request Porkbun would POST to a subscribed endpoint.
func deliveryRequest(t *testing.T, body string, sign bool) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/hooks/porkbun", strings.NewReader(body))
	if sign {
		signature, timestamp := signedNow(t, body)
		req.Header.Set(WebhookSignatureHeader, signature)
		req.Header.Set(WebhookTimestampHeader, timestamp)
		req.Header.Set(WebhookEventHeader, "domain.renewed")
		req.Header.Set(WebhookIDHeader, "018f9c2a-7b3e-7c41-9b8a-2f1e6d4c5a90")
	}
	return req
}

func TestParseWebhook(t *testing.T) {
	req := deliveryRequest(t, testWebhookBody, true)

	event, err := ParseWebhook(req, testWebhookSecret)

	require.NoError(t, err)
	assert.Equal(t, WebhookEventDomainRenewed, event.Event)
	assert.Equal(t, "018f9c2a-7b3e-7c41-9b8a-2f1e6d4c5a90", event.ID)
	assert.Equal(t, 2026, event.CreatedAt.Year())

	// The event-specific payload is left raw for the caller to decode.
	var data struct {
		Domain     string `json:"domain"`
		TLD        string `json:"tld"`
		ExpireDate string `json:"expireDate"`
	}
	require.NoError(t, json.Unmarshal(event.Data, &data))
	assert.Equal(t, "example.com", data.Domain)
	assert.Equal(t, "2027-07-27 22:09:58", data.ExpireDate)
}

// The body is restored so a handler can still log or re-read the raw bytes.
func TestParseWebhook_BodyStillReadable(t *testing.T) {
	req := deliveryRequest(t, testWebhookBody, true)

	_, err := ParseWebhook(req, testWebhookSecret)
	require.NoError(t, err)

	rest, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.JSONEq(t, testWebhookBody, string(rest))
}

func TestParseWebhook_Rejects(t *testing.T) {
	t.Run("unsigned request", func(t *testing.T) {
		_, err := ParseWebhook(deliveryRequest(t, testWebhookBody, false), testWebhookSecret)
		assert.ErrorIs(t, err, ErrWebhookNoSignature)
	})

	t.Run("wrong secret", func(t *testing.T) {
		_, err := ParseWebhook(deliveryRequest(t, testWebhookBody, true), "whsec_someoneelse")
		assert.ErrorIs(t, err, ErrWebhookBadSignature)
	})

	t.Run("valid signature over a non-json body", func(t *testing.T) {
		_, err := ParseWebhook(deliveryRequest(t, "not json", true), testWebhookSecret)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decoding webhook payload")
	})
}

// End to end through a real handler, the way a consumer wires it up.
func TestParseWebhook_InHandler(t *testing.T) {
	var got *WebhookEvent

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		event, err := ParseWebhook(r, testWebhookSecret)
		if err != nil {
			http.Error(w, "invalid webhook", http.StatusBadRequest)
			return
		}
		got = event
		w.WriteHeader(http.StatusOK)
	})

	for _, tt := range []struct {
		name     string
		sign     bool
		wantCode int
	}{
		{name: "signed delivery is accepted", sign: true, wantCode: http.StatusOK},
		{name: "unsigned delivery is rejected", sign: false, wantCode: http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got = nil
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, deliveryRequest(t, testWebhookBody, tt.sign))

			assert.Equal(t, tt.wantCode, rec.Code)
			if tt.wantCode == http.StatusOK {
				require.NotNil(t, got)
				assert.Equal(t, WebhookEventDomainRenewed, got.Event)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestBaseResponse_ResponseHeaders(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/account/balance", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-API-Version", "3.9")
		w.Header().Set("Idempotent-Replayed", "true")
		w.Header().Set("X-RateLimit-Limit", "20")
		w.Header().Set("X-RateLimit-Remaining", "19")
		w.Header().Set("X-RateLimit-Reset", "3600")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"SUCCESS","balance":1234,"display":"$12.34"}`)
	})

	resp, err := client.Account.GetBalance(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "3.9", resp.APIVersion())
	assert.True(t, resp.IdempotentReplayed())

	limits, ok := resp.RateLimit()
	require.True(t, ok)
	assert.Equal(t, int64(20), limits.Limit)
	assert.Equal(t, int64(19), limits.Remaining)
	assert.Equal(t, int64(3600), limits.Reset)
}

// Most endpoints send no rate limit headers, and a zero-value response has no
// HTTP response at all: neither may panic.
func TestBaseResponse_ResponseHeaders_Absent(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	handleFixture(t, "/account/balance", "GET", "/account/balance-success.http")

	resp, err := client.Account.GetBalance(context.Background())
	require.NoError(t, err)

	_, ok := resp.RateLimit()
	assert.False(t, ok)
	assert.False(t, resp.IdempotentReplayed())

	var zero BaseResponse
	_, ok = zero.RateLimit()
	assert.False(t, ok)
	assert.Empty(t, zero.APIVersion())
	assert.False(t, zero.IdempotentReplayed())
}
