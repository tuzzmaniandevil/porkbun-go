# Changelog

## v1.1.0 - 2026-07-30

Covers the whole of the [Porkbun API v3.9 specification](https://porkbun.com/api/json/v3/spec):
all 53 paths, with every documented request, query and response field modelled.

Upgrading from v1.0.2 is mostly a no-op at the call site. See
[Breaking changes](#breaking-changes) for the handful that need an edit, and
[Behaviour changes](#behaviour-changes) for the ones that compile but act
differently.

### Added

**Domains.** `CheckDomain`, `CreateDomain`, `RenewDomain`, `TransferDomain`,
`GetTransfer`, `ListTransfers`, `GetDomain`, `UpdateAutoRenew`,
`GetRegistrationRequirements`, and glue records (`GetGlueRecords`,
`CreateGlueRecord`, `UpdateGlueRecord`, `DeleteGlueRecord`). `ListDomains` gained
the v3.1 filters (`Domain`, `NameContains`, `Tlds`, `ExpiringWithinDays`,
`AutoRenew`, `ApiAccess`, `SortName`, `SortDirection`) and returns `Count`.

**New services.** `client.Account` (balance, API spend settings, registration
invites), `client.Marketplace`, `client.Email`, and `client.Webhooks` (create,
update, rotate secret, test, delete, list deliveries, resend).

**Webhook delivery verification.** `ParseWebhook` and `VerifyWebhookSignature`
check the HMAC-SHA256 signature in constant time and reject replays outside
`DefaultWebhookTolerance`. `SignWebhook` is exported for tests and fixtures.

**API key authorization.** `client.ApiKey.Request` / `client.ApiKey.Retrieve`,
including the v3.9 PKCE flow, with `NewPKCECodeVerifier` and `PKCECodeChallenge`.

**Per-call request options.** `WithDryRun()` rehearses a write without performing
it; `WithIdempotencyKey(key)` sends the `Idempotency-Key` header so a retry
cannot double-charge.

The API implements `dryRun` on `CreateDomain`, `RenewDomain`, `TransferDomain`,
`UpdateNameServers` and the DNS record writes. A dry run is always
distinguishable from a real write: the domain operations return the
`DryRunPreview` fields, including the `Cost` a live call would charge, and the
DNS writes return `DryRun` and `WouldSucceed`.

**Typed vocabularies.** Every enumerated value in the specification has a Go
constant: `WebhookStatus`, `DeliveryStatus`, `SortDirection`, `DomainSortField`,
`MarketplaceSortField`, `AutoRenewStatus`, `YesNo`, `CloudflareStatus`,
`InviteState`, `DeliveryMode`, `DryRunOperation`, `TermsAgreement`, and
`StatusSuccess` / `StatusError` / `StatusPending` for `BaseResponse.Status`,
alongside the existing `ErrorCode`, `NextActionType`, `WebhookEventType`,
`DnsRecordType` and `TransferStatus`. Enum fields take the constant directly;
their zero value means "unset". `Ptr(v)` returns a pointer to any value, for the
optional fields that need one:

```go
opts := &porkbun.DomainListOptions{
    SortName:           porkbun.DomainSortByExpireDate,
    SortDirection:      porkbun.SortAscending,
    ExpiringWithinDays: porkbun.Ptr(int64(30)),
    AutoRenew:          porkbun.Yes,
}
```

`YesNo` is a separate type from `AutoRenewStatus` because the two vocabularies
differ: the auto-renew *filter* on `ListDomains` takes `yes`/`no`, while
`UpdateAutoRenew` takes `on`/`off`.

**Errors.** Responses carry `Code` and `NextAction` (typed as `ErrorCode` and
`NextActionType`, with constants for every documented value), plus `RequestID`
for support and log correlation.

**Client options.** `Options.BaseURL` points the client at a mock, a proxy or a
recording harness. The HTTP client `NewClient` builds when `Options.HttpClient`
is nil carries `DefaultTimeout` (30s), since the zero `http.Client` waits
forever; a supplied client is left exactly as it came.

**Other.** `IP()`, TLD filtering on `ListPricing`, header authentication
(`X-API-Key`) so the GET-only endpoints work, `SSHFP` record type, `redirectType`
and masked URL forwards (v3.8), `cloudflare` on DNS retrieval, `apiAccess` on
domains, `PubKey` on DNSSEC records, `notes` on DNS edits, and the
`APIVersion()` / `RateLimit()` / `IdempotentReplayed()` response accessors.

### Fixed

Some of these are in code paths that predate this release and so affect anyone
upgrading from v1.0.2; others are in features added during this cycle and never
shipped broken. Both are listed, because from a reader's point of view the
question is what the code does now, not when it started doing it.

#### Writes that could act against intent

- **`WithDryRun()` cannot perform the write it was asked to rehearse.** The flag
  is sent only to the endpoints that implement it, and `WithDryRun()` on any
  other call returns an error without sending a request. Previously it went out
  on every endpoint, and because the API discards an unknown field and carries
  on, rehearsing a `DeleteGlueRecord`, `Webhooks.Delete` or `Email.SetPassword`
  performed it.
- **The `byType` endpoints unqualify their subdomain.** `GetRecordsByType`,
  `EditRecordByType` and `DeleteRecordByType` carry it in the path, and passed a
  fully qualified name straight through, addressing `/A/www.example.com`, which
  matches no record. A delete would remove nothing and report success.
- **`CreateRecord` omits the record id, and `CreateRecord` / `EditRecord`
  unqualify the record name.** Retrieval reports `www.example.com` while the
  writes take the subdomain alone and qualify it themselves, so a name carried
  over unchanged created `www.example.com.example.com`. The suffix is stripped
  case-insensitively, and the domain itself becomes `""`, the root record. A
  record read from `GetRecords` can now be passed straight back to
  `CreateRecord`.
- **A 200 carrying `{"status":"ERROR"}` is returned as an error.** The
  specification permits this on 19 operations, including the DNS and glue writes,
  where it previously came back with `err == nil`.
- **`AddDomainUrlForward` sends a body the API accepts.** `includePath` and
  `wildcard` default to `"no"` rather than the empty string; `type` is derived
  from `RedirectType` when only the latter is set (301 permanent, 302 and 307
  temporary, masked masked); and `subdomain` is unqualified, since the API takes
  only alphanumerics and hyphens there.
- **`UpdateAutoRenew` keeps the trailing slash** on the bulk-only call,
  `/domain/updateAutoRenew/`. That is the documented form; the API serves an HTML
  404 for the path without it.
- **`AddURLForward` no longer posts an empty `type`.** With neither `Type` nor
  `RedirectType` set, the derivation fell through and sent `"type":""` for a
  required field, which the API rejects. It now defaults to `temporary`, the
  API's own default, as `includePath` and `wildcard` already did.
- **The glue writes unqualify their subdomain.** `GetGlueRecords` reports the full
  hostname while the writes take the host portion alone, so the read-modify-write
  loop addressed `/domain/updateGlue/example.com/ns1.example.com`.
- **`GlueRecord` round-trips.** It had an `UnmarshalJSON` and no `MarshalJSON`, so
  re-encoding a decoded response emitted Go field names that its own unmarshaller
  then rejected.
- **One malformed date no longer costs a whole page.** `Domain.UnmarshalJSON`
  returned an error for any date it could not parse, which aborted the enclosing
  array decode: a single bad row in a 1000-domain `ListDomains` returned a
  silently truncated list. An unparsable date now reads as the zero time, which
  `IsZero` detects.
- **The webhook replay window rejects absurd timestamps.** `time.Since` saturates
  at `math.MinInt64` for a far-future instant, and negating that is a no-op, so
  the drift check passed anything more than ~292 years ahead. It compares whole
  unix seconds now. A zero tolerance applies `DefaultWebhookTolerance` instead of
  disabling the check, so the forgotten argument is the safe one; pass a negative
  tolerance to disable it.
- **`ParseWebhook` does not panic on a request with a nil `Body`.**
- **Path segments are escaped, and a relative one is refused.** A `/` in a domain
  or subdomain becomes `%2F` instead of being interpolated raw, where it could
  retarget the call. A segment of `.` or `..` cannot be escaped away, because
  `url.PathEscape` leaves dots alone: `..` reached the wire as
  `/dns/delete/../5`. Those are rejected before the request is sent. `*` is still
  sent as-is, since it names a wildcard DNS record and cannot act as a separator.

#### Errors and responses

- **A decode failure names the call.** A 200 carrying an HTML interstitial, the
  likeliest failure behind a gateway, produced a bare `invalid character '<'`
  with no method, URL or status code, and `errors.As` could not classify it.
  Decode and body-read failures are now wrapped with the request and the status,
  keeping `%w` so the underlying `*json.SyntaxError` stays matchable.
- **`ErrorResponse.Error()` no longer reproduces the query string.** `InviteStatus`
  carries the invite token there, and error strings end up in logs. The message
  also gained the `porkbun:` prefix the rest of the package uses, kept the status
  code in the branch that was dropping it, and lost its capitalised `Error:` lead.
- **A substituted `HTTPClient` cannot panic the SDK.** `Options.HTTPClient` exists
  to be replaced, but a double returning `(&http.Response{StatusCode: 200}, nil)`
  the shape most hand-written mocks use, nil-dereferenced inside `request`.
  Both that and a `(nil, nil)` return are errors now.
- **`RateLimit()` reports a 429 that sends only `X-RateLimit-Reset`.** It keyed off
  `X-RateLimit-Limit` alone, so the retry time the API documents was unreachable
  on exactly the response that carries it.
- **`RequestID` falls back to the `X-Request-Id` header**, so it is populated on
  the non-JSON error bodies where it is most worth reporting.
- **`HTTPResponse.Body` reads as empty rather than closed.** The field is exported
  for its status and headers, but reading the body returned `http: read on closed
  response body`.
- **Every method returns a non-nil response alongside its error.** Nine methods
  returned `nil` from their argument guards, so a caller following the documented
  contract nil-dereferenced on exactly those calls.

- **Nil arguments return an error instead of panicking.** `NewClient(nil)` is the
  documented way to build a credential-free client, and
  `UpdateNameServers(ctx, domain, nil)`, `CreateRecord`, `EditRecord`,
  `EditRecordByType`, `CreateDnssecRecord` and `AddDomainUrlForward` all reject a
  missing payload rather than dereferencing it.
- **A non-JSON error body still yields an `*ErrorResponse`.** Porkbun serves HTML
  for an unrecognised path; that used to arrive as an untyped error, putting the
  status code and `errors.As` out of reach.
- **The HTTP response is attached even when a call fails**, so `RateLimit()` and
  `APIVersion()` work on the response after a 429 and not only on the error.
- **The response carries the reported failure whichever way it arrived.**
  `resp.Status`, `resp.Code` and `resp.Message` are populated for a 4xx as well
  as for an `ERROR` inside a 200; the body is now decoded before the status code
  is judged. The error carried everything either way, so `errors.As` is
  unaffected.
- **An error reported inside a 200 takes precedence over a decode failure.** An
  error body carries an error shape, whose fields can clash with the success
  shape the response type expects; returning that decode error hid the `code` a
  caller branches on.
- **`Coupons` reports why decoding failed**, rather than masking every cause
  behind one generic type error.

#### Encoding

- **Every map-typed response field accepts the empty array the API sends for an
  empty map.** `pricing`, `records`, `results` and `coupons` all arrive as `[]`
  when they have no entries, the ordinary case for a domain without DNSSEC. Only
  `coupons` handled it; the other three failed to decode.
- **`Bool` accepts a JSON `true` / `false`** alongside the `"1"` / `1` forms.
- **`Domain` survives a round trip.** It marshals its dates in the format the API
  uses rather than the RFC 3339 a `time.Time` produces, and `ApiAccess` is no
  longer dropped when false.
- **The response-only booleans keep their value.** `dryRun` and `wouldSucceed`
  are the answer a dry run exists to deliver, and were being omitted on the way
  back out, as were the equivalents on `DryRunPreview` and
  `RegistrationRequirementsResponse`.
- **The nullable JSON Schema fields decode `null` to nil.** `RequestSchema` and
  `RegistryRequirements` are `RawJSON`, so an absent schema is absent by both
  `!= nil` and `len() > 0`. `json.RawMessage` stores an explicit `null` as four
  bytes, which read as a value that was there.
- **`CreateDnssecRecord` omits `pubKey`.** It is reported only on retrieval and
  is not a field of the create body. Submit key data through the `KeyData*`
  fields.
- **Credentials are no longer sent twice.** An authenticated POST carried them in
  the body *and* the `X-API-Key` headers. The API applies header auth only when
  the body has none, so the header copy did nothing but widen where the secret
  could be captured by a proxy or a request log.
- **DNSSEC algorithm 7 (RSASHA1-NSEC3-SHA1) is recognised.** `IsValid` reported
  false for a deployed DS algorithm.
- **`WebhookDelivery.Payload` and `WebhookEvent.Data` decode an explicit `null` to
  nil**, like the other raw-JSON fields.
- **Response-only fields keep what the API sent.** `omitempty` on
  `WebhookEndpoint.Secret`, the `createDate` fields and `GetRecordsResponse.Cloudflare`
  dropped values on re-marshal.
- **`unqualifyName` accepts an absolute name.** A trailing root dot, which DNS
  tooling writes and the API never returns, was passed through, so the API
  appended the domain a second time. The two root forms the `byType` paths
  accepted are also normalised to one.
- **Header auth requires both halves.** A client holding a public key but no
  secret sent `X-API-Key` with an empty `X-Secret-API-Key`, earning a
  `MISSING_SECRETAPIKEY` instead of the anonymous access the credential-free
  endpoints grant.

**Documentation.** A package overview for pkg.go.dev, runnable examples covering
the client, DNS writes, dry runs, idempotency keys, error inspection and webhook
verification, and a `.golangci.yml` enabling `revive`'s `var-naming` and
`exported` rules so the initialism casing cannot regress.

### Breaking changes

`apidiff` reports 75 incompatible changes against v1.0.2, alongside 323 compatible
ones. Most are mechanical: 30 identifiers renamed to fix initialism casing, and 18
request types and interfaces unexported because no exported method ever accepted
or returned them. The compiler catches every one.

**Renames.** Go wants initialisms in uniform case, and the package was already
inconsistent: it spelled `TLD`, `HTTPResponse`, `RequestID` and `AuthURL`
correctly while spelling `Dns`, `Ssl`, `Apikey`, `Url` and `Id` wrongly. **JSON
tags are unchanged**, so nothing about the wire format moves.

| Old | New |
| --- | --- |
| `Client.Dns`, `Client.Ssl`, `Client.ApiKey` | `Client.DNS`, `Client.SSL`, `Client.APIKey` |
| `Options.HttpClient`, `.ApiKey`, `.SecretApiKey` | `Options.HTTPClient`, `.APIKey`, `.SecretAPIKey` |
| `DnsService`, `DnsRecord`, `DnsRecordType` | `DNSService`, `DNSRecord`, `DNSRecordType` |
| `SslService`, `SslRetrieveResponse` | `SSLService`, `SSLRetrieveResponse` |
| `SslRetrieveResponse.Certificatechain`, `.Privatekey`, `.Publickey` | `.CertificateChain`, `.PrivateKey`, `.PublicKey` |
| `ApiKeyService`, `ApiKeyRequestResponse`, `ApiKeyRetrieveResponse`, `ApiKeyRequestOptions` | `APIKeyService`, and `APIKey` in place of `ApiKey` on the rest |
| `ApiKeyRetrieveResponse.Apikey`, `.SecretApiKey` | `.APIKey`, `.SecretAPIKey` |
| `ApiSettings`, `ApiSettingsResponse`, `GetApiSettings` | `APISettings`, `APISettingsResponse`, `GetAPISettings` |
| `DnssecAlgorithm`, `DnssecDigestType`, `DnssecRecords`, `DnssecRecordData` and their 15 constants | `DNSSECAlgorithm`, `DNSSECDigestType`, `DNSSECRecords`, `DNSSECRecord`, and `DNSSEC` in place of `Dnssec` on the constants |
| `GetDnssecRecords`, `CreateDnssecRecord`, `DeleteDnssecRecord` | `GetDNSSECRecords`, `CreateDNSSECRecord`, `DeleteDNSSECRecord` |
| `UrlForward`, `UrlForwardData`, `UrlForwardData.Id` | `URLForward`, `URLForwardRecord`, `.ID` |
| `GetDomainURLForwarding`, `AddDomainUrlForward`, `DeleteDomainUrlForward` | `ListURLForwards`, `AddURLForward`, `DeleteURLForward` |
| `Domain.ApiAccess`, `DomainListOptions.ApiAccess` | `.APIAccess` |
| `DomainListOptions.Tlds`, `MarketplaceListOptions.Tlds` | `.TLDs` |
| `MarketplaceListOptions.SldLengthMin`/`Max`, `MarketplaceDomain.SldLength` | `.SLDLengthMin`/`Max`, `.SLDLength` |
| `MarketplaceSortBySldLength` | `MarketplaceSortBySLDLength` |
| `RegistrationRequirementsResponse.ApiRegisterable`, `.NotApiRegisterableReason` | `.APIRegisterable`, `.NotAPIRegisterableReason` |

**Unexported.** `BaseRequest`, `ApiKeyAcceptor`, `DryRunAcceptor` and the 30
`*Request` types were internal wire structs. No exported method took or returned
any of them, and each was constructed only inside the method that posted it.
They are now unexported. If you referenced one, you were reaching into plumbing;
the corresponding options struct or method argument is the supported route.

**Signatures and types.**

| Change | Fix |
| --- | --- |
| `GetDomain` takes `*GetDomainOptions`, not a bare `YesNo` | `GetDomain(ctx, d, &porkbun.GetDomainOptions{IncludeLabels: porkbun.Yes})`, or `nil` for the defaults |
| `UpdateAutoRenew` takes `(ctx, status, domains []string)` | One list instead of a domain plus an extras slice: `UpdateAutoRenew(ctx, porkbun.AutoRenewOn, []string{"example.com"})`. An empty list is an error |
| `GetRecords` is deprecated in favour of `ListRecords` and `GetRecord` | `ListRecords(ctx, domain)` for all, `GetRecord(ctx, domain, id)` for one. `GetRecords` still works |
| `ErrorResponse.Message` moved to the embedded `BaseResponse` | Reads (`err.Message`) are unaffected; only composite literals need `BaseResponse: BaseResponse{Message: ...}` |
| `Coupon.Amount` is `float64`; `Coupon.FirstYearOnly` is `YesNo` | The spec types them as a number and a `yes`/`no` enum. A literal still compiles; prefer `porkbun.Yes` |
| `URLForward.IncludePath` and `.Wildcard` are `YesNo`, not `string` | As above. A comparison against a `string` *variable* needs a conversion |
| `DNSRecord.ID`, `.TTL`, `.Prio` are `FlexInt64`; `EditRecord`/`EditTypeRecord` `TTL` and `Prio` are `int64` | The spec types all of them as integers. Drop the quotes: `TTL: 600`. Read an id with `record.ID.Int64()`; a record that does not exist yet has `ID == 0` rather than a nil pointer |
| `DomainListOptions.Start` is `*int64`; `.IncludeLabels` is `YesNo` | `Ptr(int64(1000))` instead of `String("1000")`; `porkbun.Yes` instead of `String("yes")` |
| `DryRunPreview.MonthlySpendLimit`, `.MonthlySpendSoFar`, `.WithinMonthlySpendLimit` are pointers | The API sends them only when a cap is configured, and a plain `false` was indistinguishable from "would breach the cap". Nil-check, or read `WouldSucceed` |
| `DNSSECRecord.MaxSigLife` is `*string` | `Ptr("3600")` |
| `RegistrationRequirementsResponse.RequestSchema`/`.RegistryRequirements`, `WebhookDelivery.Payload` and `WebhookEvent.Data` are `RawJSON`, not `json.RawMessage` | `string(...)` and `json.Unmarshal` are unchanged; a `json.RawMessage` variable needs a conversion |
| `PricingResponse.Pricing` and `GetDNSSECRecordsResponse.Records` are named map types | Assignable both ways with the plain map, so this only affects code naming the old type explicitly |
| `UpdateNameServers` takes `NameServers`, not `*NameServers` | Drop the `&`. A nil or empty list returns an error rather than being sent |
| `(*DnsRecord).UnmarshalJSON` and `DnsRecordType.ToPtr()` removed | `FlexInt64` handles the string and number forms now; nothing in the package asks for a `*string` any more |
| `DomainListOptions` is no longer comparable, having gained `TLDs []string` | Compare fields individually instead of `==` |
| Adding `opts ...RequestOption` changed 10 method *types* | Only affects code assigning a method to a variable of the old signature |
| Structs gained fields | Only affects unkeyed composite literals; use field names |

### Behaviour changes

These compile unchanged but act differently.

- `UpdateNameServersResponse` carries the dry-run verdict (`DryRun`,
  `WouldSucceed`) that `WithDryRun()` was already documented to produce.
- `ListPricing` returns a non-nil response alongside an error, matching every
  other method. Code testing `resp == nil` after a failure should test `err`.
- The client sends `X-API-Key` / `X-Secret-API-Key` on the GET endpoints, which
  have no body to carry credentials. A POST sends them in the body only. A test
  double asserting an exact header set may notice.
- The endpoints the spec marks as taking no credentials (`Ping` without keys,
  `IP`, `ListPricing`, and the `/apikey` flow) send none at all. This matters for
  `IP`: a key restricted to specific source addresses would otherwise be rejected
  from the very address a dynamic DNS client is trying to discover.
- A failed request whose body was not JSON produced `HTTP error 400: Bad Request`
  and now produces an `*ErrorResponse` whose text includes the method and URL.
  Match on `Code` or the status rather than on the string.
- A 200 carrying `{"status":"ERROR"}` is an error. Statuses that are not failures
  are unaffected: `/apikey/retrieve` still returns `PENDING` successfully, and
  `UpdateAutoRenew` still reports per-domain failures under a top-level
  `SUCCESS`.
- A `VerifyWebhookSignature` tolerance of zero now applies
  `DefaultWebhookTolerance` rather than disabling the replay check. Pass a
  negative tolerance for the old behaviour.
- A domain date the API sends in an unexpected shape reads as the zero time
  instead of failing the call.
- `errors.Is(err, porkbun.ErrCodeInsufficientFunds)` works: `ErrorCode` implements
  `error` so it can be an `errors.Is` target, alongside the existing `errors.As`
  route to the whole `*ErrorResponse`.

### Deprecated

- `BoolString` and `BoolNumber` are aliases for `Bool`, which accepts both the
  string and the number form. Existing code keeps compiling; new code should use
  `Bool`.
- `String(v)` remains, but `Ptr(v)` does the same for any type.

### Known limitations

- The 13 endpoints that accept both `GET` and `POST` are implemented with one
  verb each. No capability is missing.
- `POST /auth/login` is not implemented, along with the `Authorization: Bearer`
  scheme it issues tokens for. It is partner-only, needing `auth:login` on the
  key and a request signature, and is absent from the OpenAPI document.
- Timestamps are not uniform. `Domain.CreateDate` and `.ExpireDate` are
  `time.Time`; every other timestamp is the string the API sent, with its format
  documented on the field. The API uses at least three formats
  (`2006-01-02 15:04:05` for domains, a bare date for renewal expiry, RFC 3339 for
  webhook events), so a single shared type cannot cover them. Unifying this needs
  per-field types and is deferred to v2.
- The test suite is not parallel-safe: the mock-server helpers use package-level
  state, so adding `t.Parallel()` introduces a data race. `-race` is clean as the
  suite stands.
