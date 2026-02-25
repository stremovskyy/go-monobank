package go_monobank

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// CurrencyCode is ISO 4217 numeric code.
type CurrencyCode int32

const (
	CurrencyUAH CurrencyCode = 980
)

// PaymentType defines operation type.
//
// debit - regular debit
// hold  - hold (requires later finalize), hold lifetime ~9 days (per docs)
type PaymentType string

const (
	PaymentTypeDebit PaymentType = "debit"
	PaymentTypeHold  PaymentType = "hold"
)

// InitiationKind defines who initiated the wallet/token payment.
//
// merchant - merchant-initiated (recurring, etc)
// client   - client-initiated (customer requested)
type InitiationKind string

const (
	InitiationMerchant InitiationKind = "merchant"
	InitiationClient   InitiationKind = "client"
)

// InvoiceStatus is a payment/invoice state.
// We keep it as string to avoid over-restricting API evolution.
type InvoiceStatus string

const (
	InvoiceCreated    InvoiceStatus = "created"
	InvoiceProcessing InvoiceStatus = "processing"
	InvoiceSuccess    InvoiceStatus = "success"
	InvoiceFailure    InvoiceStatus = "failure"
	InvoiceReversed   InvoiceStatus = "reversed"
	InvoiceExpired    InvoiceStatus = "expired"
)

// IsSuccess reports whether status indicates successful payment completion.
func (s InvoiceStatus) IsSuccess() bool {
	return s == InvoiceSuccess
}

// IsFailure reports whether status indicates terminal non-success outcome.
func (s InvoiceStatus) IsFailure() bool {
	return s == InvoiceFailure || s == InvoiceReversed || s == InvoiceExpired
}

// IsPending reports whether status indicates non-final in-progress state.
func (s InvoiceStatus) IsPending() bool {
	return s == InvoiceCreated || s == InvoiceProcessing
}

// IsFinal reports whether status is final (success or non-success terminal).
func (s InvoiceStatus) IsFinal() bool {
	return s.IsSuccess() || s.IsFailure()
}

// MerchantPaymInfo mirrors docs "merchantPaymInfo" (minimal subset).
// You can extend it later without breaking callers.
type MerchantPaymInfo struct {
	Reference      string   `json:"reference,omitempty"`
	Destination    string   `json:"destination,omitempty"`
	Comment        string   `json:"comment,omitempty"`
	CustomerEmails []string `json:"customerEmails,omitempty"`

	// NOTE: docs contain more fields (discounts, basketOrder, etc).
	// They are intentionally omitted from the minimal SDK.
}

type SaveCardData struct {
	SaveCard bool   `json:"saveCard"`
	WalletID string `json:"walletId,omitempty"`
}

// InvoiceCreateResponse is returned by POST /api/merchant/invoice/create.
type InvoiceCreateResponse struct {
	InvoiceID string `json:"invoiceId"`
	PageURL   string `json:"pageUrl"`
}

// ParsedPageURL parses response pageUrl as an absolute URL.
func (r *InvoiceCreateResponse) ParsedPageURL() (*url.URL, error) {
	if r == nil {
		return nil, fmt.Errorf("invoice create response is nil")
	}
	parsed, err := url.Parse(strings.TrimSpace(r.PageURL))
	if err != nil {
		return nil, fmt.Errorf("invoice create response: cannot parse pageUrl %q: %w", r.PageURL, err)
	}
	if !parsed.IsAbs() {
		return nil, fmt.Errorf("invoice create response: pageUrl is not absolute: %q", r.PageURL)
	}
	return parsed, nil
}

// WalletPaymentResponse is returned by POST /api/merchant/wallet/payment.
// Also resembles some other payment-related responses.
type WalletPaymentResponse struct {
	InvoiceID     string        `json:"invoiceId"`
	TDSURL        *string       `json:"tdsUrl,omitempty"`
	Status        InvoiceStatus `json:"status"`
	FailureReason *string       `json:"failureReason,omitempty"`
	Amount        int64         `json:"amount"`
	Currency      CurrencyCode  `json:"ccy"`
	CreatedDate   time.Time     `json:"createdDate"`
	ModifiedDate  time.Time     `json:"modifiedDate"`
}

// IsSuccess reports whether wallet payment status is successful.
func (r *WalletPaymentResponse) IsSuccess() bool {
	return r != nil && r.Status.IsSuccess()
}

// IsFailure reports whether wallet payment status is final non-success.
func (r *WalletPaymentResponse) IsFailure() bool {
	return r != nil && r.Status.IsFailure()
}

// IsPending reports whether wallet payment status is not final yet.
func (r *WalletPaymentResponse) IsPending() bool {
	return r != nil && r.Status.IsPending()
}

// IsFinal reports whether wallet payment status is final.
func (r *WalletPaymentResponse) IsFinal() bool {
	return r != nil && r.Status.IsFinal()
}

// Requires3DS reports whether response contains non-empty tdsUrl.
func (r *WalletPaymentResponse) Requires3DS() bool {
	return r != nil && r.TDSURL != nil && strings.TrimSpace(*r.TDSURL) != ""
}

// InvoiceStatusResponse is returned by GET /api/merchant/invoice/status
// and is also the webhook payload body.
type InvoiceStatusResponse struct {
	InvoiceID     string        `json:"invoiceId"`
	Status        InvoiceStatus `json:"status"`
	FailureReason *string       `json:"failureReason,omitempty"`
	ErrCode       *string       `json:"errCode,omitempty"`

	Amount      int64        `json:"amount"`
	Currency    CurrencyCode `json:"ccy"`
	FinalAmount *int64       `json:"finalAmount,omitempty"`

	CreatedDate  time.Time `json:"createdDate"`
	ModifiedDate time.Time `json:"modifiedDate"`

	Reference   *string `json:"reference,omitempty"`
	Destination *string `json:"destination,omitempty"`

	CancelList  []CancelItem `json:"cancelList,omitempty"`
	PaymentInfo *PaymentInfo `json:"paymentInfo,omitempty"`
	WalletData  *WalletData  `json:"walletData,omitempty"`
	TipsInfo    *TipsInfo    `json:"tipsInfo,omitempty"`
}

// IsSuccess reports whether invoice status is successful.
func (r *InvoiceStatusResponse) IsSuccess() bool {
	return r != nil && r.Status.IsSuccess()
}

// IsFailure reports whether invoice status is final non-success.
func (r *InvoiceStatusResponse) IsFailure() bool {
	return r != nil && r.Status.IsFailure()
}

// IsPending reports whether invoice status is not final yet.
func (r *InvoiceStatusResponse) IsPending() bool {
	return r != nil && r.Status.IsPending()
}

// IsFinal reports whether invoice status is final.
func (r *InvoiceStatusResponse) IsFinal() bool {
	return r != nil && r.Status.IsFinal()
}

type CancelItem struct {
	Status       InvoiceStatus `json:"status"`
	Amount       int64         `json:"amount"`
	Currency     CurrencyCode  `json:"ccy"`
	CreatedDate  time.Time     `json:"createdDate"`
	ModifiedDate time.Time     `json:"modifiedDate"`

	ApprovalCode *string `json:"approvalCode,omitempty"`
	RRN          *string `json:"rrn,omitempty"`
	ExtRef       *string `json:"extRef,omitempty"`

	MaskedPan *string `json:"maskedPan,omitempty"`
}

type PaymentInfo struct {
	MaskedPan     *string `json:"maskedPan,omitempty"`
	ApprovalCode  *string `json:"approvalCode,omitempty"`
	RRN           *string `json:"rrn,omitempty"`
	TranID        *string `json:"tranId,omitempty"`
	Terminal      *string `json:"terminal,omitempty"`
	PaymentSystem *string `json:"paymentSystem,omitempty"`
	PaymentMethod *string `json:"paymentMethod,omitempty"`
	Fee           *int64  `json:"fee,omitempty"`
}

type WalletData struct {
	CardToken string `json:"cardToken"`
	WalletID  string `json:"walletId"`
	Status    string `json:"status"`
}

type TipsInfo struct {
	EmployeeID *string `json:"employeeId,omitempty"`
	Amount     *int64  `json:"amount,omitempty"`
}

// PublicKeyResponse is returned by GET /api/merchant/pubkey.
// The value is a base64-encoded PEM public key (per docs examples).
type PublicKeyResponse struct {
	Key string `json:"key"`
}
