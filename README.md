# Porkbun Go SDK

[![Go Report Card](https://goreportcard.com/badge/github.com/tuzzmaniandevil/porkbun-go)](https://goreportcard.com/report/github.com/tuzzmaniandevil/porkbun-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/tuzzmaniandevil/porkbun-go.svg)](https://pkg.go.dev/github.com/tuzzmaniandevil/porkbun-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Go client for the [Porkbun API](https://porkbun.com/api/json/v3/documentation).
It covers all 53 paths in the [v3.9 specification](https://porkbun.com/api/json/v3/spec):
domains, DNS and DNSSEC, SSL certificates, URL forwarding, glue records,
nameservers, account settings, the marketplace, email passwords, webhooks, and the
API key authorization flow.

## What it gives you beyond the raw endpoints

- `WithDryRun()` rehearses a write without performing it. Passing it to an endpoint
  the API does not implement it on returns an error instead of sending the request,
  because the API discards an unknown flag and performs the write anyway.
- `WithIdempotencyKey()` sends an `Idempotency-Key`, so a retry after a network
  failure replays the original response instead of charging twice.
- Machine-readable `ErrorCode` on every failure, matchable with `errors.Is`, plus
  the `NextAction` remediation hint the API supplies.
- `RateLimit()`, `APIVersion()` and `IdempotentReplayed()` on every response,
  working after a failed call as well as a successful one.
- Tolerant scalar types for the fields the API encodes inconsistently: `Bool`
  (`true`, `"1"` or `1`), `FlexInt64` (a number or a quoted number), `YesNo`, and
  `RawJSON` (an explicit `null` decodes to nil).
- A Go constant for every enumerated value in the specification.
- `Options.BaseURL` for pointing the client at a test server or proxy, and a 30s
  `DefaultTimeout` on the HTTP client the SDK builds when you do not supply one.
- `IP()` for a credential-free public-IP lookup. Use it for dynamic DNS: a key
  restricted by source address would be rejected from the address you are trying
  to discover.
- `GetRegistrationRequirements` to find out whether and how a TLD can be
  registered through the API before you try.
- `ParseWebhook` to verify a delivery's signature before you act on it.

## Installation

```bash
go get github.com/tuzzmaniandevil/porkbun-go
```

Requires Go 1.21 or later.

## Usage

### Basic Usage

Listing the domains in your account:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/tuzzmaniandevil/porkbun-go"
)

func main() {
    client := porkbun.NewClient(&porkbun.Options{
        APIKey:       "your_api_key",
        SecretAPIKey: "your_secret_api_key",
    })

    resp, err := client.Domains.ListDomains(context.Background(), &porkbun.DomainListOptions{})
    if err != nil {
        log.Fatalf("Error listing domains: %v", err)
    }

    for _, domain := range resp.Domains {
        fmt.Println(domain.Domain)
    }
}
```

### Registering a domain

Quote the price first, then register with the cost in pennies. Pass
`porkbun.WithDryRun()` to run every pre-flight check without charging anything, and
`porkbun.WithIdempotencyKey(key)` so a retry after a network blip cannot double-charge.
Both are per-call request options; they never carry over to a later call.

```go
check, err := client.Domains.CheckDomain(ctx, "example.com")
if err != nil {
    log.Fatal(err)
}

if check.Response.Avail != porkbun.Yes {
    log.Fatal("not available")
}

preview, err := client.Domains.CreateDomain(ctx, "example.com", &porkbun.CreateDomainOptions{
    Cost:         973, // pennies; check.Response.Price is the same figure in dollars, "9.73"
    AgreeToTerms: porkbun.TermsAgreed,
}, porkbun.WithDryRun())
if err != nil {
    log.Fatal(err)
}

fmt.Println(preview.WouldSucceed, preview.CostDisplay)
```

Drop `porkbun.WithDryRun()` to register for real, and add
`porkbun.WithIdempotencyKey("<unique-id>")` so a retried call cannot charge twice.

The API implements `dryRun` on `CreateDomain`, `RenewDomain`, `TransferDomain`,
`UpdateNameServers` and the DNS record writes. Passing `WithDryRun()` to any
other call returns an error rather than sending it: the API discards a flag it
does not know and performs the write, so a rehearsal of `DeleteGlueRecord` would
otherwise delete the record.

### Receiving webhooks

Register an endpoint, then verify every delivery before acting on it.
`ParseWebhook` checks the HMAC-SHA256 signature in constant time and rejects
deliveries whose timestamp has drifted more than five minutes, so a captured
request cannot be replayed later.

```go
// Once, at setup. Store the secret securely: it is the signing key.
endpoint, err := client.Webhooks.Create(ctx, "https://example.org/hooks/porkbun",
    []porkbun.WebhookEventType{porkbun.WebhookEventAll})
if err != nil {
    log.Fatal(err)
}
endpointSecret := endpoint.Endpoint.Secret

// In your handler.
http.HandleFunc("/hooks/porkbun", func(w http.ResponseWriter, r *http.Request) {
    event, err := porkbun.ParseWebhook(r, endpointSecret)
    if err != nil {
        http.Error(w, "invalid webhook", http.StatusBadRequest)
        return
    }

    switch event.Event {
    case porkbun.WebhookEventDomainExpiring:
        var data struct {
            Domain     string `json:"domain"`
            ExpireDate string `json:"expireDate"`
        }
        _ = json.Unmarshal(event.Data, &data)
        log.Printf("%s expires %s", data.Domain, data.ExpireDate)
    }

    w.WriteHeader(http.StatusOK) // any 2xx acknowledges the delivery
})
```

Deduplicate on `event.ID`, because an endpoint may receive the same event more than
once, and a resend deliberately reuses the original id. Use
`porkbun.VerifyWebhookSignature` directly if you need a custom tolerance or
already have the raw body in hand. A zero tolerance there applies the five-minute
default, and only a negative one disables the replay check.

`client.Webhooks.EventTypes(ctx)` returns the live catalog of subscribable events,
which is worth preferring to the `porkbun.WebhookEvent*` constants if you want to
pick up types added after this release.

### Error handling

Every error the API itself reports is an `*porkbun.ErrorResponse` carrying a
machine-readable `Code` and, where the API knows how to recover, a `NextAction`.
Transport failures and invalid arguments are returned as ordinary errors, so test
with `errors.As` rather than a type assertion. `NextAction` is optional: check it
before dereferencing.

Match a code directly with `errors.Is`:

```go
if errors.Is(err, porkbun.ErrCodeInsufficientFunds) {
    // top up and retry
}
```

or take the whole response with `errors.As` when you want the remediation hint:

```go
var apiErr *porkbun.ErrorResponse
if errors.As(err, &apiErr) && apiErr.NextAction != nil {
    fmt.Println(apiErr.NextAction.Hint) // "Add account credit and retry."
}
```

Every method returns a non-nil response alongside its error, so inspecting the
response after checking `err` is safe. On a transport failure only `HTTPResponse`
is populated; on a failure the API reported, `Status`, `Code` and `Message` are
too, whether it arrived as a 4xx or as `{"status":"ERROR"}` inside a 200.

The client does not retry. Wrap calls in your own backoff, and pass
`WithIdempotencyKey` to any write you intend to retry.

`Code` and `NextAction.Type` are typed against the documented vocabularies
(`porkbun.ErrCode*`, `porkbun.NextAction*`), but both remain plain strings
underneath, so an unrecognised value from a newer API version still compares and
prints normally.

### Advanced Usage

For a runnable end-to-end example, see [cmd/main.go](https://github.com/tuzzmaniandevil/porkbun-go/blob/main/cmd/main.go), which pings the API and walks every domain's DNS and DNSSEC records.

## Documentation

The API reference is on [pkg.go.dev](https://pkg.go.dev/github.com/tuzzmaniandevil/porkbun-go).

## Testing

Run the tests using `go test`:

```bash
go test ./...
```

The tests are hermetic: they run against `httptest` servers and recorded fixtures under `fixtures/`, so no API credentials and no network access are required.

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request. For major changes, please open an issue first to discuss what you would like to change.

### Guidelines

- Write clear, concise commit messages.
- Ensure all tests pass before submitting a pull request.
- Follow the existing code style and format your code with `gofmt`.
- Run `golangci-lint run ./...`; the committed config enables `revive`'s `var-naming` and `exported` rules, which keep initialism casing (`API`, `DNS`, `URL`, `ID`) consistent.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgements

- [Porkbun](https://porkbun.com) for documenting the API properly, including a machine-readable spec.
- [Go](https://golang.org) community for tools and inspiration.
