package porkbun

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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

	assert.NoError(t, err)

	assert.Contains(t, data, "apikey")
	assert.Contains(t, data, "secretapikey")

	assert.Equal(t, "1234", data["apikey"])
	assert.Equal(t, "5678", data["secretapikey"])
}

func getRequestJSON(r *http.Request) (map[string]interface{}, error) {
	var data map[string]interface{}

	// Read the body and put a copy back into the request for subsiquent reads
	body, _ := io.ReadAll(r.Body)
	r.Body.Close() //  must close
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func testRequestJSON(t *testing.T, r *http.Request, values map[string]interface{}) {
	data, err := getRequestJSON(r)

	assert.NoError(t, err)
	assert.Equal(t, data, values)
}

func testErrorResponse(t *testing.T, err error) {
	assert.Error(t, err)
	assert.IsType(t, &ErrorResponse{}, err)

	errResponse, ok := err.(*ErrorResponse)
	assert.True(t, ok)
	assert.Equal(t, "ERROR", errResponse.Status)
	assert.NotEmpty(t, errResponse.Error())
}

func httpResponseFixture(t *testing.T, filename string) *http.Response {
	data, err := os.ReadFile("./fixtures" + filename)
	assert.NoError(t, err)

	// http.ReadResponse does not like it when there is a `Transfer-Encoding: chunked` header present, So we just remove it
	data = bytes.ReplaceAll(data, []byte("Transfer-Encoding: chunked\n"), []byte{})
	data = bytes.ReplaceAll(data, []byte("Transfer-Encoding: chunked\r\n"), []byte{})

	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(data)), nil)
	assert.NoError(t, err)

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

func TestErrorResponse_ErrorMessage(t *testing.T) {
	errResp := ErrorResponse{
		BaseResponse: BaseResponse{
			Status: "ERROR",
		},
		Message: "An error occurred",
	}

	assert.Equal(t, "Error: An error occurred", errResp.Error())
}
