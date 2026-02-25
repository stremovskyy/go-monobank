# go-monobank

[![Go Reference](https://pkg.go.dev/badge/github.com/stremovskyy/go-monobank.svg)](https://pkg.go.dev/github.com/stremovskyy/go-monobank)
[![Go Report Card](https://goreportcard.com/badge/github.com/stremovskyy/go-monobank)](https://goreportcard.com/report/github.com/stremovskyy/go-monobank)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

Minimal, production-focused Go SDK for [monobank Acquiring API](https://api.monobank.ua/docs/acquiring.html).

This library follows the same style as other `stremovskyy/*` SDKs:
- simple public interface
- fluent request builder
- explicit typed errors
- practical defaults for production

## Features

- Card tokenization (verification flow): `POST /api/merchant/invoice/create`
- Payment by saved card token: `POST /api/merchant/wallet/payment`
- Invoice status lookup: `GET /api/merchant/invoice/status`
- Webhook parsing and signature verification (`X-Sign`, ECDSA SHA-256)
- Structured API and transport errors with `errors.Is(...)` support
- Business-level payment error helpers (`PaymentError`)
- Dry-run mode for safe payload inspection
- Built-in log levels (`None/Error/Warning/Info/Debug/All`)

## Requirements

- Go `1.23+`

## Installation

```bash
go get github.com/stremovskyy/go-monobank@latest
```

## Supported API Flows

| SDK method | Endpoint | Purpose |
|---|---|---|
| `Verification` | `POST /api/merchant/invoice/create` | Create invoice + enable `saveCardData` tokenization |
| `VerificationLink` | `POST /api/merchant/invoice/create` | Convenience helper that returns parsed `pageUrl` |
| `Payment` | `POST /api/merchant/wallet/payment` | Charge by `cardToken` |
| `Status` | `GET /api/merchant/invoice/status` | Fetch current invoice state |
| `PublicKey` | `GET /api/merchant/pubkey` | Fetch webhook verification key |
| `ParseWebhook` | N/A | Parse webhook JSON body |
| `VerifyWebhook` | N/A | Verify `X-Sign` against raw body |
| `ParseAndVerifyWebhook` | N/A | Verify signature + parse payload |

## Quick Start

```go
package main

import (
	"fmt"
	"log"
	"os"

	go_monobank "github.com/stremovskyy/go-monobank"
	sdklog "github.com/stremovskyy/go-monobank/log"
)

func main() {
	token := os.Getenv("MONO_TOKEN")
	if token == "" {
		log.Fatal("set MONO_TOKEN")
	}

	client := go_monobank.NewClient(
		go_monobank.WithToken(token),
	)
	client.SetLogLevel(sdklog.LevelInfo)

	resp, err := client.Verification(
		go_monobank.NewRequest().
			WithAmount(100). // 1.00 UAH in minor units
			WithCurrency(go_monobank.CurrencyUAH).
			WithRedirectURL("https://example.com/return").
			WithWebHookURL("https://example.com/webhook").
			WithReference("verify-card-001").
			WithDestination("Card verification").
			SaveCard("customer-wallet-123"),
	)
	if err != nil {
		log.Fatal(err)
	}

	payURL, err := resp.ParsedPageURL()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("invoiceId:", resp.InvoiceID)
	fmt.Println("pageUrl:", payURL.String())
}
```

## Important: `Verification` vs `VerificationLink`

`VerificationLink(request)` internally calls `Verification(request)`.

That means if you call both methods one after another, you create **two separate invoices** and perform **two API requests**.

Best practice:
- Use `Verification(...)` when you need `invoiceId` and `pageUrl`.
- Use `VerificationLink(...)` only when you need parsed `*url.URL` and do not need the original response object.

Convenience helper:
- If you already called `Verification(...)`, use `resp.ParsedPageURL()` to parse/validate `pageUrl` without a second API request.

## Payment by Card Token

```go
resp, err := client.Payment(
	go_monobank.NewRequest().
		WithToken(token). // optional if WithToken(...) is set on client
		WithCardToken(cardToken).
		WithAmount(4200).
		WithCurrency(go_monobank.CurrencyUAH).
		WithInitiationKind(go_monobank.InitiationMerchant).
		WithReference("wallet-pay-001").
		WithDestination("Token payment"),
)
if err != nil {
	return err
}

if pe := resp.PaymentError(); pe != nil {
	// business/payment failure (not transport failure)
	fmt.Println(pe.Error())
}
```

## Status and Business Error Inspection

```go
status, err := client.Status(
	go_monobank.NewRequest().
		WithInvoiceID(invoiceID),
)
if err != nil {
	return err
}

if pe := status.PaymentError(); pe != nil {
	fmt.Println("payment failed:", pe)
	for _, m := range pe.Metas {
		fmt.Printf("code=%s text=%s contact=%s\n", m.Code, m.Text, m.Contact)
	}
}

if status.IsFinal() && status.IsSuccess() {
	fmt.Println("payment completed")
}
```

## Webhook Verification

```go
func handler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}

	xSign := r.Header.Get("X-Sign")
	event, err := client.ParseAndVerifyWebhook(body, xSign)
	if err != nil {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	_ = event // process event safely
	w.WriteHeader(http.StatusOK)
}
```

Webhook key resolution order:
1. `WithWebhookPublicKeyPEM(...)`
2. `WithWebhookPublicKeyBase64(...)`
3. fetch from `/api/merchant/pubkey` using default token

## Configuration Options

- `WithToken(token)` sets default `X-Token`.
- `WithBaseURL(url)` overrides base URL.
- `WithTimeout(d)` sets request timeout.
- `WithKeepAlive(d)` sets transport keepalive.
- `WithMaxIdleConns(n)` sets HTTP max idle connections.
- `WithIdleConnTimeout(d)` sets idle connection timeout.
- `WithClient(*http.Client)` injects custom HTTP client.
- `WithWebhookPublicKeyBase64(key)` sets webhook key (base64 PEM).
- `WithWebhookPublicKeyPEM(pemBytes)` sets webhook key (raw PEM).

## Logging

Set SDK log level via:

```go
client.SetLogLevel(sdklog.LevelInfo)
```

Available levels:
- `LevelNone`
- `LevelError`
- `LevelWarning`
- `LevelInfo`
- `LevelDebug`
- `LevelAll`

Backward-compatible aliases are also available: `Off`, `Error`, `Warn`, `Info`, `Debug`.

## Dry Run

Dry run skips the outgoing HTTP request and lets you inspect endpoint/payload.

```go
_, _ = client.Verification(
	go_monobank.NewRequest().
		WithAmount(100).
		WithCurrency(go_monobank.CurrencyUAH).
		SaveCard("wallet-id"),
	go_monobank.DryRun(func(endpoint string, payload any) {
		fmt.Println("endpoint:", endpoint)
		fmt.Printf("payload: %#v\n", payload)
	}),
)
```

If you call `DryRun()` without a handler, payload is printed through the SDK logger at `Info` level.

## Error Handling

Use standard `errors.Is(...)` / `errors.As(...)` patterns.

```go
if err != nil {
	switch {
	case errors.Is(err, go_monobank.ErrValidation):
		// local request validation issue
	case errors.Is(err, go_monobank.ErrTransport):
		// network/TLS/timeout issue
	case errors.Is(err, go_monobank.ErrRateLimited):
		var apiErr *go_monobank.APIError
		if errors.As(err, &apiErr) && apiErr.RetryAfter != nil {
			fmt.Println("retry after:", apiErr.RetryAfter)
		}
	default:
		// fallback
	}
}
```

Common sentinel errors:
- `ErrValidation`
- `ErrEncode`
- `ErrTransport`
- `ErrDecode`
- `ErrBadRequest`
- `ErrInvalidToken`
- `ErrNotFound`
- `ErrMethodNotAllowed`
- `ErrRateLimited`
- `ErrServerError`
- `ErrUnexpectedResponse`
- `ErrInvalidSignature`
- `ErrPaymentError`

## Production Best Practices

- Use client-level `WithToken(...)` to avoid repetitive token wiring.
- Do not call `Verification(...)` and `VerificationLink(...)` for the same checkout session.
- Verify webhook signature against raw bytes before business processing.
- Treat `LevelDebug` logs as sensitive in production (payloads may include tokenized payment data).
- Handle `429` with `Retry-After` backoff.
- Keep `reference` values unique in your own system for better reconciliation.

## Examples

Run from repository root:

```bash
MONO_TOKEN=... go run ./examples/verification
MONO_TOKEN=... CARD_TOKEN=... go run ./examples/payment_by_token
MONO_TOKEN=... INVOICE_ID=... go run ./examples/status
MONO_TOKEN=... go run ./examples/webhook_http
```

## Contributing

Issues and pull requests are welcome.

## License

MIT. See [LICENSE](./LICENSE).
