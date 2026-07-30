package porkbun

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	mux    *http.ServeMux
	client *Client
	server *httptest.Server
)

func setupMockServer(creds bool) {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	client = NewClient(&Options{})

	if creds {
		client.apiKey = "1234"
		client.secret = "5678"
	}

	client.baseURL = server.URL
}

func teardownMockServer() {
	server.Close()
}

func testMethod(t *testing.T, r *http.Request, want string) {
	assert.Equal(t, want, r.Method)
}

func testHeaders(t *testing.T, r *http.Request) {
	assert.Equal(t, "application/json", r.Header.Get("Accept"))
	assert.Equal(t, defaultUserAgent, r.Header.Get("User-Agent"))
}

func testCredentials(t *testing.T, r *http.Request) {
	data, err := getRequestJSON(r)

	require.NoError(t, err)

	assert.Contains(t, data, "apikey")
	assert.Contains(t, data, "secretapikey")

	assert.Equal(t, "1234", data["apikey"])
	assert.Equal(t, "5678", data["secretapikey"])
}

// testHeaderCredentials asserts the API key headers used by the GET endpoints.
func testHeaderCredentials(t *testing.T, r *http.Request) {
	t.Helper()

	assert.Equal(t, "1234", r.Header.Get("X-API-Key"))
	assert.Equal(t, "5678", r.Header.Get("X-Secret-API-Key"))
}

// fixtureHandler replays a fixture, asserting the method and the credentials
// appropriate to it: body credentials on POST, header credentials on GET.
// assertRequest, when supplied, receives the decoded POST body.
func fixtureHandler(t *testing.T, method string, fixture string, assertRequest func(map[string]interface{})) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, fixture)
		assert.NotNil(t, httpResponse)

		testMethod(t, r, method)
		testHeaders(t, r)

		if method == http.MethodGet {
			testHeaderCredentials(t, r)
		} else {
			testCredentials(t, r)
		}

		if assertRequest != nil {
			data, err := getRequestJSON(r)
			require.NoError(t, err)
			assertRequest(data)
		}

		writeFixture(w, httpResponse)
	}
}

// writeFixture replays a fixture's headers, status and body, so the response
// headers the SDK reads (X-API-Version, the rate limit set, Idempotent-Replayed)
// reach the client exactly as the API sent them.
func writeFixture(w http.ResponseWriter, fixture *http.Response) {
	for name, values := range fixture.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	w.WriteHeader(fixture.StatusCode)
	_, _ = io.Copy(w, fixture.Body)
}

// handleFixture registers a fixture-backed handler for an authenticated endpoint.
func handleFixture(t *testing.T, path string, method string, fixture string) {
	t.Helper()

	mux.HandleFunc(path, fixtureHandler(t, method, fixture, nil))
}

// handleFixtureRequest is handleFixture with an assertion on the request body.
func handleFixtureRequest(t *testing.T, path string, method string, fixture string, assertRequest func(map[string]interface{})) {
	t.Helper()

	mux.HandleFunc(path, fixtureHandler(t, method, fixture, assertRequest))
}

// handleFixtureNoAuth registers a fixture-backed handler for an endpoint the API
// documents as taking no credentials, and asserts that none are sent: not in the
// body, and not as the API key headers. Use it with setupMockServer(true) to
// prove the suppression, rather than relying on a client that has no keys.
func handleFixtureNoAuth(t *testing.T, path string, method string, fixture string, assertRequest func(map[string]interface{})) {
	t.Helper()

	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		httpResponse := httpResponseFixture(t, fixture)

		testMethod(t, r, method)
		testHeaders(t, r)
		testNoCredentials(t, r)

		if assertRequest != nil {
			data, err := getRequestJSON(r)
			assert.NoError(t, err)
			assertRequest(data)
		}

		writeFixture(w, httpResponse)
	})
}

// testNoCredentials asserts the request carries no credentials at all, in either
// the headers or the body. A GET has no body to check.
func testNoCredentials(t *testing.T, r *http.Request) {
	t.Helper()

	assert.Empty(t, r.Header.Get("X-API-Key"), "X-API-Key must not be sent to a credential-free endpoint")
	assert.Empty(t, r.Header.Get("X-Secret-API-Key"), "X-Secret-API-Key must not be sent to a credential-free endpoint")

	if r.Method == http.MethodGet {
		return
	}

	data, err := getRequestJSON(r)
	if !assert.NoError(t, err) {
		return
	}
	assert.NotContains(t, data, "apikey", "apikey must not be sent to a credential-free endpoint")
	assert.NotContains(t, data, "secretapikey", "secretapikey must not be sent to a credential-free endpoint")
}

func getRequestJSON(r *http.Request) (map[string]interface{}, error) {
	var data map[string]interface{}

	// Read the body and put a copy back into the request for subsequent reads
	body, _ := io.ReadAll(r.Body)
	_ = r.Body.Close() // must close before replacing
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func testRequestJSON(t *testing.T, r *http.Request, values map[string]interface{}) {
	data, err := getRequestJSON(r)

	require.NoError(t, err)
	assert.Equal(t, data, values)
}

func testErrorResponse(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)

	// require, not assert: a regression that returns a different error type must
	// fail this test rather than nil-panic and abort the whole package.
	var errResponse *ErrorResponse
	require.ErrorAs(t, err, &errResponse)

	assert.Equal(t, "ERROR", errResponse.Status)
	assert.NotEmpty(t, errResponse.Error())
}

func httpResponseFixture(t *testing.T, filename string) *http.Response {
	data, err := os.ReadFile("./fixtures" + filename)
	require.NoError(t, err)

	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(data)), nil)
	require.NoError(t, err)

	return resp
}

func TestStringPointerWithEmptyString(t *testing.T) {
	str := String("")
	assert.NotNil(t, str)
	assert.Equal(t, "", *str)
}

func TestBoolStringWithInvalidValue(t *testing.T) {
	var bs BoolString
	err := json.Unmarshal([]byte(`"invalid"`), &bs)

	assert.Error(t, err)
}

func TestBoolNumberWithInvalidValue(t *testing.T) {
	var bn BoolNumber
	err := json.Unmarshal([]byte(`"invalid"`), &bn)

	assert.Error(t, err)
}

func TestUnmarshalBool(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    bool
		wantErr assert.ErrorAssertionFunc
	}{
		{name: "valid string false", data: []byte(`"false"`), want: false, wantErr: assert.NoError},
		{name: "valid string true", data: []byte(`"true"`), want: true, wantErr: assert.NoError},
		{name: "valid string 0", data: []byte(`"0"`), want: false, wantErr: assert.NoError},
		{name: "valid string 1", data: []byte(`"1"`), want: true, wantErr: assert.NoError},
		{name: "empty string", data: []byte(`""`), want: false, wantErr: assert.NoError},
		{name: "invalid string", data: []byte(`"invalid"`), want: false, wantErr: assert.Error},
		{name: "valid number 0", data: []byte(`0`), want: false, wantErr: assert.NoError},
		{name: "valid number 1", data: []byte(`1`), want: true, wantErr: assert.NoError},
		{name: "invalid number", data: []byte(`2`), want: false, wantErr: assert.Error},
		{name: "json true", data: []byte(`true`), want: true, wantErr: assert.NoError},
		{name: "json false", data: []byte(`false`), want: false, wantErr: assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmarshalBool(tt.data)
			if !tt.wantErr(t, err, fmt.Sprintf("unmarshalBool(%v)", tt.data)) {
				return
			}
			assert.Equalf(t, tt.want, got, "unmarshalBool(%v)", tt.data)
		})
	}
}

func TestErrorResponse_ErrorMessage(t *testing.T) {
	errResp := ErrorResponse{
		BaseResponse: BaseResponse{
			Status:  "ERROR",
			Message: "an error occurred",
		},
	}

	assert.Equal(t, "porkbun: an error occurred", errResp.Error())
}

// The query carries the invite token on InviteStatus, and an error string ends up
// in logs, so Error must not reproduce it.
func TestErrorResponse_ErrorRedactsTheQuery(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet,
		"https://api.porkbun.com/api/json/v3/account/inviteStatus?token=inv_SECRET", nil)
	require.NoError(t, err)

	errResp := &ErrorResponse{BaseResponse: BaseResponse{
		Status:  StatusError,
		Message: "Bad token.",
		HTTPResponse: &http.Response{
			StatusCode: http.StatusBadRequest,
			Request:    request,
		},
	}}

	got := errResp.Error()
	assert.NotContains(t, got, "inv_SECRET")
	assert.NotContains(t, got, "token=")
	assert.Equal(t,
		"porkbun: GET https://api.porkbun.com/api/json/v3/account/inviteStatus: 400: Bad token.",
		got)
}

// A status code is still reported when the response carries no request.
func TestErrorResponse_ErrorWithoutARequest(t *testing.T) {
	errResp := &ErrorResponse{BaseResponse: BaseResponse{
		Message:      "Slow down.",
		HTTPResponse: &http.Response{StatusCode: http.StatusTooManyRequests},
	}}

	assert.Equal(t, "porkbun: 429: Slow down.", errResp.Error())
}

// An ErrorCode doubles as an errors.Is target, so a caller can match one without
// unwrapping the response.
func TestErrorResponse_IsMatchesAnErrorCode(t *testing.T) {
	err := error(&ErrorResponse{BaseResponse: BaseResponse{
		Status: StatusError,
		Code:   ErrCodeInsufficientFunds,
	}})

	assert.True(t, errors.Is(err, ErrCodeInsufficientFunds))
	assert.False(t, errors.Is(err, ErrCodeRateLimitExceeded))
	assert.True(t, errors.Is(fmt.Errorf("renewing: %w", err), ErrCodeInsufficientFunds))

	// A response with no code must not match every code.
	assert.False(t, errors.Is(&ErrorResponse{}, ErrorCode("")))
}

func TestFlexInt64(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    FlexInt64
		wantErr assert.ErrorAssertionFunc
	}{
		{name: "number", data: []byte(`7891021`), want: 7891021, wantErr: assert.NoError},
		{name: "string", data: []byte(`"7891021"`), want: 7891021, wantErr: assert.NoError},
		{name: "negative string", data: []byte(`"-5"`), want: -5, wantErr: assert.NoError},
		{name: "empty string", data: []byte(`""`), want: 0, wantErr: assert.NoError},
		{name: "null", data: []byte(`null`), want: 0, wantErr: assert.NoError},
		{name: "not a number", data: []byte(`"abc"`), want: 0, wantErr: assert.Error},
		{name: "object", data: []byte(`{}`), want: 0, wantErr: assert.Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got FlexInt64
			if !tt.wantErr(t, json.Unmarshal(tt.data, &got)) {
				return
			}
			assert.Equal(t, tt.want, got)
			assert.Equal(t, int64(tt.want), got.Int64())
		})
	}
}

func TestErrorResponse_CodeAndNextAction(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/create/example.com", fixtureHandler(t, "POST", "/domains/create-insufficientfunds.http", nil))

	_, err := client.Domains.CreateDomain(context.Background(), "example.com", &CreateDomainOptions{
		Cost:         973,
		AgreeToTerms: "yes",
	})

	testErrorResponse(t, err)

	errResponse, ok := err.(*ErrorResponse)
	assert.True(t, ok)
	assert.Equal(t, ErrCodeInsufficientFunds, errResponse.Code)
	assert.NotNil(t, errResponse.NextAction)
	assert.Equal(t, NextActionAddFunds, errResponse.NextAction.Type)
	assert.Equal(t, "https://porkbun.com/account/billing", errResponse.NextAction.URL)
}

func TestErrorResponse_RateLimit(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	mux.HandleFunc("/domain/checkDomain/example.com", fixtureHandler(t, "POST", "/domains/checkDomain-ratelimited.http", nil))

	_, err := client.Domains.CheckDomain(context.Background(), "example.com")

	testErrorResponse(t, err)

	errResponse, ok := err.(*ErrorResponse)
	assert.True(t, ok)
	assert.Equal(t, ErrCodeRateLimitExceeded, errResponse.Code)
	assert.Equal(t, int64(7), errResponse.TTLRemaining)
}

// json.RawMessage stores an explicit null as the four bytes "null", which both a
// nil check and len() read as a value that is present.
func TestRawJSON_NullDecodesAsAbsent(t *testing.T) {
	var resp RegistrationRequirementsResponse
	require.NoError(t, json.Unmarshal([]byte(
		`{"status":"SUCCESS","tld":"com","registryRequirements":null,"requestSchema":null}`), &resp))

	assert.Nil(t, resp.RegistryRequirements)
	assert.Empty(t, resp.RegistryRequirements)
	assert.Nil(t, resp.RequestSchema)

	// A value present is kept verbatim.
	const schema = `{"type":"object","required":["nexusCategory"]}`
	require.NoError(t, json.Unmarshal([]byte(
		`{"status":"SUCCESS","tld":"us","registryRequirements":`+schema+`}`), &resp))
	assert.JSONEq(t, schema, string(resp.RegistryRequirements))

	// And marshals back as itself, an absent one as null.
	out, err := json.Marshal(RawJSON(schema))
	require.NoError(t, err)
	assert.JSONEq(t, schema, string(out))

	out, err = json.Marshal(RawJSON(nil))
	require.NoError(t, err)
	assert.Equal(t, "null", string(out))
}

// ErrorCode implements error only so it can be an errors.Is target; Error returns
// the wire code itself.
func TestErrorCode_Error(t *testing.T) {
	assert.Equal(t, "INSUFFICIENT_FUNDS", ErrCodeInsufficientFunds.Error())
	assert.Equal(t, "", ErrorCode("").Error())
}
