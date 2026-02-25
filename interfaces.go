package go_monobank

import (
	"net/url"

	"github.com/stremovskyy/go-monobank/log"
)

// Monobank is the main SDK interface.
//
// Minimal supported flows:
//   - Verification (invoice/create + saveCardData)
//   - Payment by card token (wallet/payment)
//   - Status (invoice/status)
//   - Webhook parsing + signature verification (X-Sign)
type Monobank interface {
	// Verification creates an invoice with saveCardData (tokenization).
	// It returns invoiceId + pageUrl.
	Verification(request *Request, opts ...RunOption) (*InvoiceCreateResponse, error)
	// VerificationLink is a helper that returns only pageUrl as parsed *url.URL.
	VerificationLink(request *Request, opts ...RunOption) (*url.URL, error)

	// Payment performs a charge by tokenized card (wallet/payment).
	Payment(request *Request, opts ...RunOption) (*WalletPaymentResponse, error)

	// Status returns current invoice status (invoice/status).
	Status(request *Request, opts ...RunOption) (*InvoiceStatusResponse, error)

	// PublicKey fetches merchant webhook verification public key (pubkey).
	PublicKey(request *Request, opts ...RunOption) (*PublicKeyResponse, error)

	// ParseWebhook parses webhook JSON body.
	ParseWebhook(body []byte) (*InvoiceStatusResponse, error)
	// VerifyWebhook verifies X-Sign signature against raw body.
	VerifyWebhook(body []byte, xSign string) error
	// ParseAndVerifyWebhook is a convenience method.
	ParseAndVerifyWebhook(body []byte, xSign string) (*InvoiceStatusResponse, error)

	// SetLogLevel changes SDK logging level.
	SetLogLevel(level log.Level)
}
