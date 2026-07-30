package porkbun

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRequestOptionsReachTheWire calls every method that accepts RequestOptions
// and asserts the options actually influence the request that goes out.
//
// A method can declare opts and then call post without forwarding them, which
// compiles and discards the caller's idempotency key in silence. The methods are
// walked by reflection so a new one is covered without being remembered.
func TestRequestOptionsReachTheWire(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	var (
		mu   sync.Mutex
		seen []string
	)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Get("Idempotency-Key"))
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"SUCCESS"}`))
	})

	targets := map[string]any{
		"Client":      client,
		"Domains":     client.Domains,
		"DNS":         client.DNS,
		"SSL":         client.SSL,
		"Account":     client.Account,
		"Marketplace": client.Marketplace,
		"Email":       client.Email,
		"Webhooks":    client.Webhooks,
		"Pricing":     client.Pricing,
		"APIKey":      client.APIKey,
	}

	checked := 0

	for _, name := range sortedKeys(targets) {
		value := reflect.ValueOf(targets[name])

		for i := 0; i < value.NumMethod(); i++ {
			method := value.Type().Method(i)
			if !acceptsRequestOptions(method.Type) {
				continue
			}

			label := name + "." + method.Name
			key := "idem-" + label

			t.Run(label, func(t *testing.T) {
				mu.Lock()
				seen = nil
				mu.Unlock()

				args := []reflect.Value{value, reflect.ValueOf(context.Background())}
				// Every parameter between the context and the options.
				for p := 2; p < method.Type.NumIn()-1; p++ {
					args = append(args, sampleValue(method.Type.In(p)))
				}
				args = append(args, reflect.ValueOf(WithIdempotencyKey(key)))

				method.Func.Call(args)

				mu.Lock()
				got := append([]string(nil), seen...)
				mu.Unlock()

				require.NotEmpty(t, got, "%s never reached the server, so the sample arguments are wrong", label)
				assert.Equal(t, []string{key}, got,
					"%s accepts RequestOptions but did not forward them, so WithIdempotencyKey and WithDryRun are silently ignored", label)
			})

			checked++
		}
	}

	// Guards the reflection itself: renaming or unexporting the option parameter
	// would leave this test passing while checking nothing.
	assert.GreaterOrEqual(t, checked, 20, "expected to find the methods that take RequestOptions")
}

// TestWithDryRunNeverPerformsTheWrite calls every method that accepts
// RequestOptions with WithDryRun, and asserts each one either sends the flag or
// refuses the call outright.
//
// The API discards a dryRun it does not implement and performs the write, so a
// method that accepts the option without its endpoint supporting it would delete
// the thing the caller asked to rehearse. The methods are walked by reflection so
// a new one is covered without being remembered.
func TestWithDryRunNeverPerformsTheWrite(t *testing.T) {
	setupMockServer(true)
	defer teardownMockServer()

	var (
		mu   sync.Mutex
		sent []bool
	)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, ok := body["dryRun"]

		mu.Lock()
		sent = append(sent, ok)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"SUCCESS"}`))
	})

	// The nine endpoints the specification documents dryRun on: the three
	// billable operations, the DNS record writes, and the nameserver update.
	supported := map[string]bool{
		"DNS.CreateRecord":          true,
		"DNS.DeleteRecord":          true,
		"DNS.DeleteRecordByType":    true,
		"DNS.EditRecord":            true,
		"DNS.EditRecordByType":      true,
		"Domains.CreateDomain":      true,
		"Domains.RenewDomain":       true,
		"Domains.TransferDomain":    true,
		"Domains.UpdateNameServers": true,
	}

	targets := map[string]any{
		"Client":      client,
		"Domains":     client.Domains,
		"DNS":         client.DNS,
		"SSL":         client.SSL,
		"Account":     client.Account,
		"Marketplace": client.Marketplace,
		"Email":       client.Email,
		"Webhooks":    client.Webhooks,
		"Pricing":     client.Pricing,
		"APIKey":      client.APIKey,
	}

	checked := 0

	for _, name := range sortedKeys(targets) {
		value := reflect.ValueOf(targets[name])

		for i := 0; i < value.NumMethod(); i++ {
			method := value.Type().Method(i)
			if !acceptsRequestOptions(method.Type) {
				continue
			}

			label := name + "." + method.Name

			t.Run(label, func(t *testing.T) {
				mu.Lock()
				sent = nil
				mu.Unlock()

				args := []reflect.Value{value, reflect.ValueOf(context.Background())}
				for p := 2; p < method.Type.NumIn()-1; p++ {
					args = append(args, sampleValue(method.Type.In(p)))
				}
				args = append(args, reflect.ValueOf(WithDryRun()))

				results := method.Func.Call(args)
				err, _ := results[len(results)-1].Interface().(error)

				mu.Lock()
				got := append([]bool(nil), sent...)
				mu.Unlock()

				if supported[label] {
					require.NoError(t, err, "%s supports dryRun and must not refuse it", label)
					assert.Equal(t, []bool{true}, got,
						"%s supports dryRun but sent no flag, so the write was performed for real", label)
					return
				}

				require.Error(t, err,
					"%s does not support dryRun, so it must refuse the call rather than perform the write", label)
				assert.Empty(t, got,
					"%s reached the API with a dry run it cannot honour, so the write was performed for real", label)
			})

			checked++
		}
	}

	assert.GreaterOrEqual(t, checked, 20, "expected to find the methods that take RequestOptions")
}

// acceptsRequestOptions reports whether a method's last parameter is ...RequestOption.
func acceptsRequestOptions(t reflect.Type) bool {
	if !t.IsVariadic() || t.NumIn() < 2 {
		return false
	}
	last := t.In(t.NumIn() - 1)
	return last.Kind() == reflect.Slice && last.Elem() == reflect.TypeOf(RequestOption(nil))
}

// sampleValue builds a usable argument of the given type. Values only have to be
// accepted by the client, not by the real API: the mock answers every path.
func sampleValue(t reflect.Type) reflect.Value {
	switch t.Kind() {
	case reflect.String:
		v := reflect.New(t).Elem()
		v.SetString("sample")
		return v
	case reflect.Bool:
		v := reflect.New(t).Elem()
		v.SetBool(true)
		return v
	case reflect.Int, reflect.Int64:
		v := reflect.New(t).Elem()
		v.SetInt(1)
		return v
	case reflect.Slice:
		s := reflect.MakeSlice(t, 1, 1)
		s.Index(0).Set(sampleValue(t.Elem()))
		return s
	case reflect.Pointer:
		p := reflect.New(t.Elem())
		p.Elem().Set(sampleValue(t.Elem()))
		return p
	case reflect.Struct:
		v := reflect.New(t).Elem()
		for i := 0; i < t.NumField(); i++ {
			if f := t.Field(i); f.PkgPath == "" {
				v.Field(i).Set(sampleValue(f.Type))
			}
		}
		return v
	default:
		return reflect.Zero(t)
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// A handful of keys, so an insertion sort keeps the test dependency-free.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && strings.Compare(keys[j-1], keys[j]) > 0; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}
