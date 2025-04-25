package porkbun

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
)

const (
	Version          = "1.0.0"
	defaultBaseURL   = "https://api.porkbun.com/api/json/v3"
	ipv4OnlyBaseURL  = "https://api-ipv4.porkbun.com/api/json/v3"
	defaultUserAgent = "porkbun-go/" + Version
)

// ApiKeyAcceptor defines an interface for setting API credentials.
type ApiKeyAcceptor interface {
	SetCredentials(apiKey string, secretApiKey string)
}

// BaseRequest represents the base structure for all API requests, including credentials.
type BaseRequest struct {
	SecretApiKey string `json:"secretapikey"` // The secret API key provided by Porkbun.
	Apikey       string `json:"apikey"`       // The public API key provided by Porkbun.
}

// SetCredentials sets the API and secret keys for the request.
func (br *BaseRequest) SetCredentials(apiKey string, secretApiKey string) {
	br.Apikey = apiKey
	br.SecretApiKey = secretApiKey
}

// BaseResponse represents the base structure for all API responses.
type BaseResponse struct {
	HTTPResponse *http.Response // The underlying HTTP Response.
	Status       string         `json:"status"` // Status indicating whether the command was successfully processed.
}

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	BaseResponse
	Message string `json:"message,omitempty"` // The error message provided by the API.
}

// Error implements the error interface for ErrorResponse.
func (r *ErrorResponse) Error() string {
	if r.HTTPResponse == nil || r.HTTPResponse.Request == nil {
		return fmt.Sprintf("Error: %v", r.Message)
	}
	return fmt.Sprintf("%v %v: %v %v",
		r.HTTPResponse.Request.Method, r.HTTPResponse.Request.URL,
		r.HTTPResponse.StatusCode, r.Message)
}

// String is a helper function that allocates a new string value and returns a pointer to it.
func String(v string) *string { return &v }

// unmarshalBool converts JSON data into a boolean value.
//
// The Porkbun API represents booleans in two formats:
// 1. Strings: "1" for true, "0" or "" for false.
// 2. Numbers: 1 for true, 0 for false.
// This function handles both formats and returns the appropriate boolean value.
func unmarshalBool(data []byte) (bool, error) {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		if str == "" {
			return false, nil
		}
		parsedBool, err := strconv.ParseBool(str)
		if err != nil {
			return false, fmt.Errorf("error parsing boolean: %s", str)
		}
		return parsedBool, nil
	}

	var num int
	if err := json.Unmarshal(data, &num); err == nil {
		switch num {
		case 0:
			return false, nil
		case 1:
			return true, nil
		default:
			return false, fmt.Errorf("error parsing boolean: %d", num)
		}
	}

	return false, fmt.Errorf("error invalid boolean format: %s", string(data))
}

// BoolString is a custom type for handling boolean values represented as "1"/"0" strings in JSON.
type BoolString bool

// UnmarshalJSON implements custom unmarshalling logic for BoolString.
func (b *BoolString) UnmarshalJSON(data []byte) error {
	parsedBool, err := unmarshalBool(data)
	if err != nil {
		return err
	}
	*b = BoolString(parsedBool)
	return nil
}

// BoolNumber is a custom type for handling boolean values represented as 1/0 numbers in JSON.
type BoolNumber bool

// UnmarshalJSON implements custom unmarshalling logic for BoolNumber.
func (b *BoolNumber) UnmarshalJSON(data []byte) error {
	parsedBool, err := unmarshalBool(data)
	if err != nil {
		return err
	}
	*b = BoolNumber(parsedBool)
	return nil
}

// buildPathSegments appends valid segments to the base path string.
// It handles both pointer and non-pointer types and skips nil values.
func buildPathSegments(base *string, segments ...any) {
	for _, v := range segments {
		if v == nil {
			continue // Skip nil values
		}

		val := reflect.ValueOf(v)
		if val.Kind() == reflect.Ptr {
			if val.IsNil() {
				continue // Skip nil pointers
			}
			// Dereference the pointer and use the value
			v = val.Elem().Interface()
		}

		*base += fmt.Sprintf("/%v", v)
	}
}
