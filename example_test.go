package porkbun_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tuzzmaniandevil/porkbun-go"
)

func Example() {
	client := porkbun.NewClient(&porkbun.Options{
		APIKey:       os.Getenv("PORKBUN_API_KEY"),
		SecretAPIKey: os.Getenv("PORKBUN_API_SECRET"),
	})

	resp, err := client.Ping(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.YourIP)
}

// A client with no credentials can still reach the public endpoints.
func ExampleNewClient_withoutCredentials() {
	client := porkbun.NewClient(nil)

	resp, err := client.Pricing.ListPricing(context.Background(), "com")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Pricing["com"].Registration)
}

func ExampleDNSService_CreateRecord() {
	client := porkbun.NewClient(&porkbun.Options{APIKey: "pk1_...", SecretAPIKey: "sk1_..."})

	resp, err := client.DNS.CreateRecord(context.Background(), "example.com", &porkbun.DNSRecord{
		Name:    "www", // the subdomain alone; the API adds the domain
		Type:    porkbun.A,
		Content: "203.0.113.10",
		TTL:     600,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.ID.Int64())
}

// A record read back from ListRecords can be passed straight to a write: the
// SDK strips the domain the API added, and drops the id the API allocated.
func ExampleDNSService_ListRecords() {
	client := porkbun.NewClient(&porkbun.Options{APIKey: "pk1_...", SecretAPIKey: "sk1_..."})
	ctx := context.Background()

	resp, err := client.DNS.ListRecords(ctx, "example.com")
	if err != nil {
		log.Fatal(err)
	}

	for _, record := range resp.Records {
		if record.Type != porkbun.A {
			continue
		}

		_, err := client.DNS.EditRecord(ctx, "example.com", record.ID.Int64(), &porkbun.EditRecord{
			Name:    record.Name,
			Type:    record.Type,
			Content: "203.0.113.11",
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}

// WithDryRun runs every pre-flight check and reports what would happen, without
// charging or creating anything.
func ExampleWithDryRun() {
	client := porkbun.NewClient(&porkbun.Options{APIKey: "pk1_...", SecretAPIKey: "sk1_..."})

	resp, err := client.Domains.CreateDomain(context.Background(), "example.com",
		&porkbun.CreateDomainOptions{
			Cost:         1099, // pennies, and must equal the price CheckDomain quoted
			AgreeToTerms: porkbun.TermsAgreed,
		},
		porkbun.WithDryRun(),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.WouldSucceed, resp.CostDisplay)
}

// WithIdempotencyKey makes a retry after a network failure replay the original
// response instead of charging twice.
func ExampleWithIdempotencyKey() {
	client := porkbun.NewClient(&porkbun.Options{APIKey: "pk1_...", SecretAPIKey: "sk1_..."})

	resp, err := client.Domains.RenewDomain(context.Background(), "example.com", 1099,
		porkbun.WithIdempotencyKey("renew-example.com-2026-Q1"))
	if err != nil {
		log.Fatal(err)
	}

	if resp.IdempotentReplayed() {
		fmt.Println("the original renewal was replayed; nothing was charged again")
	}
}

// A failure the API reported carries a machine-readable code, matchable with
// errors.Is. errors.As reaches the whole response, including the remediation
// hint the API supplies.
func ExampleErrorResponse() {
	client := porkbun.NewClient(&porkbun.Options{APIKey: "pk1_...", SecretAPIKey: "sk1_..."})

	_, err := client.Domains.CreateDomain(context.Background(), "example.com",
		&porkbun.CreateDomainOptions{Cost: 1099, AgreeToTerms: porkbun.TermsAgreed})

	switch {
	case err == nil:
		fmt.Println("registered")
	case errors.Is(err, porkbun.ErrCodeInsufficientFunds):
		fmt.Println("top up the account and retry")
	case errors.Is(err, porkbun.ErrCodeRateLimitExceeded):
		var apiErr *porkbun.ErrorResponse
		if errors.As(err, &apiErr) {
			fmt.Printf("retry in %d seconds\n", apiErr.TTLRemaining)
		}
	default:
		log.Fatal(err)
	}
}

// ParseWebhook verifies the signature before returning the event. An error means
// the delivery is not trustworthy: reject it and do not act on the payload.
func ExampleParseWebhook() {
	// The signing secret comes from Webhooks.Create or Webhooks.RotateSecret,
	// at resp.Endpoint.Secret.
	endpointSecret := os.Getenv("PORKBUN_WEBHOOK_SECRET")

	http.HandleFunc("/hooks/porkbun", func(w http.ResponseWriter, r *http.Request) {
		event, err := porkbun.ParseWebhook(r, endpointSecret)
		if err != nil {
			http.Error(w, "invalid webhook", http.StatusBadRequest)
			return
		}

		// Deduplicate on event.ID: an endpoint may see the same event twice, and
		// a resend deliberately reuses the original id.
		switch event.Event {
		case porkbun.WebhookEventDomainExpiring:
			log.Printf("expiring: %s", event.Data)
		case porkbun.WebhookEventDNSRecordDeleted:
			log.Printf("record deleted: %s", event.Data)
		}

		w.WriteHeader(http.StatusOK)
	})
}
