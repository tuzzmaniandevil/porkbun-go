// Package porkbun is a client for the Porkbun API v3.9.
//
// It covers all 53 documented paths: domain registration, renewal, transfer and
// listing; DNS records and DNSSEC; SSL certificate bundles; URL forwarding; glue
// records; nameservers; account balance and spend settings; the marketplace;
// email hosting passwords; webhook endpoints and deliveries; and the API key
// authorization flow.
//
// # Getting started
//
// A client is a set of services hanging off [Client].
//
//	client := porkbun.NewClient(&porkbun.Options{
//		APIKey:       "pk1_...",
//		SecretAPIKey: "sk1_...",
//	})
//
//	resp, err := client.DNS.ListRecords(ctx, "example.com")
//
// The zero [Client] is not usable; build one with [NewClient]. Pass nil, or the
// zero [Options], for a client with no credentials. That covers [Client.Ping],
// [Client.IP] and [PricingService.ListPricing], and the [APIKeyService] flow
// that mints credentials in the first place. Header auth needs both halves, so a
// client holding a key but no secret is refused by the API rather than treated
// as anonymous.
//
// [Options.BaseURL] points the client at a test server, a proxy or a recording
// harness. When [Options.HTTPClient] is nil the client builds one carrying
// [DefaultTimeout]; the per-call context deadline still applies, and a shorter
// one wins.
//
// # Errors
//
// Every method returns a non-nil response alongside any error, so a caller that
// inspects the response after checking err cannot nil-dereference. On a
// transport failure only HTTPResponse is populated; on a failure the API
// reported, Status, Code and Message are too, whether it arrived as a 4xx or as
// status ERROR inside a 200.
//
// A failure the API reported is an [*ErrorResponse] carrying a machine-readable
// [ErrorCode] and, where the API knows how to recover, a [NextAction]. Match a
// code with errors.Is, or take the whole response with errors.As:
//
//	if errors.Is(err, porkbun.ErrCodeInsufficientFunds) {
//		// top up and retry
//	}
//
//	var apiErr *porkbun.ErrorResponse
//	if errors.As(err, &apiErr) && apiErr.NextAction != nil {
//		log.Printf("%s: %s", apiErr.NextAction.Type, apiErr.NextAction.Hint)
//	}
//
// A 200 carrying {"status":"ERROR"} is an error too: the specification permits
// it on 19 operations, including the DNS and glue writes, where treating it as
// success would have a caller believe a record was deleted when it was not.
// StatusPending is not a failure: [APIKeyService.Retrieve] returns it while
// authorization is outstanding.
//
// The client does not retry. A failed request comes back as-is; wrap the call in
// your own backoff, and pass [WithIdempotencyKey] to any write you intend to
// retry so a repeat cannot charge twice. A request body is replayable, so a
// retrying [HTTPClient] works too.
//
// # Per-call options
//
// [WithDryRun] rehearses a write without performing it. The API implements it on
// [DomainsService.CreateDomain], [DomainsService.RenewDomain],
// [DomainsService.TransferDomain], [DomainsService.UpdateNameServers] and the
// DNS record writes; passing it anywhere else returns an error instead of
// sending the request, because the API discards a flag it does not recognise and
// performs the write anyway.
//
// # Response metadata
//
// Every response embeds [BaseResponse], which exposes the underlying
// *http.Response plus [BaseResponse.RateLimit], [BaseResponse.APIVersion] and
// [BaseResponse.IdempotentReplayed]. These work after a failed call as well as a
// successful one. The response body has already been read, so HTTPResponse.Body
// reads as empty.
//
// # Scalar types
//
// The API is inconsistent about how it encodes scalars, so this package uses
// tolerant types rather than making callers handle both forms: [Bool] accepts
// true, "1" and 1; [FlexInt64] accepts a number or a quoted number; [YesNo] is
// the "yes"/"no" vocabulary used by the listing filters and URL forward options;
// and [RawJSON] decodes an explicit null to nil. [Ptr] returns a pointer to any
// value, for the optional fields that take one.
//
// Timestamps are not yet uniform. [Domain.CreateDate] and [Domain.ExpireDate]
// are time.Time, parsed from the "2006-01-02 15:04:05" form the API uses; a
// value in any other shape reads as the zero time rather than failing the whole
// response. Every other timestamp is delivered as the string the API sent, and
// its format is documented on the field.
//
// # Webhooks
//
// [ParseWebhook] verifies a delivery's HMAC-SHA256 signature in constant time
// and rejects timestamps drifted further than [DefaultWebhookTolerance], so a
// captured request cannot be replayed later. Use
// [VerifyWebhookSignature] directly for a custom window or a body you already
// hold. Deduplicate on the event id: an endpoint may see the same event more
// than once, and a resend deliberately reuses the original id.
package porkbun
