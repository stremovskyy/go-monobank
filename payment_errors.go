package go_monobank

import (
	"fmt"
	"sort"
	"strings"
)

const (
	// PaymentErrorContactIssuingBank means customer should contact the card issuing bank.
	PaymentErrorContactIssuingBank = "issuing bank"
	// PaymentErrorContactMonobank means merchant should contact monobank support.
	PaymentErrorContactMonobank = "monobank support"
	// PaymentErrorContactCustomer means customer action is required.
	PaymentErrorContactCustomer = "customer"
	// PaymentErrorContactAPI means integrator/merchant API configuration should be checked.
	PaymentErrorContactAPI = "api/integration team"
)

// PaymentErrorMeta is a human-friendly description for a payment errCode.
// Source: monobank acquiring docs (payment errors).
// Note: failureReason from API/webhook is still the most precise explanation.
type PaymentErrorMeta struct {
	Code    string
	Text    string
	Contact string
}

// HandlingHint returns a practical next step based on contact target.
func (m PaymentErrorMeta) HandlingHint() string {
	contact := strings.ToLower(strings.TrimSpace(m.Contact))
	switch contact {
	case strings.ToLower(PaymentErrorContactIssuingBank):
		return "Ask customer to contact the issuing bank and verify card restrictions/limits."
	case strings.ToLower(PaymentErrorContactMonobank):
		return "Contact monobank support and provide invoiceId + errCode for investigation."
	case strings.ToLower(PaymentErrorContactCustomer):
		return "Ask customer to fix input/payment details and retry the payment flow."
	case strings.ToLower(PaymentErrorContactAPI):
		return "Review integration/request validation and merchant configuration in API settings."
	default:
		return "Review errCode and failureReason, then route to the responsible support team."
	}
}

// PaymentError is a business-level error parsed from webhook/status response fields:
//   - errCode
//   - failureReason
//
// It is NOT an HTTP/transport error; those are represented by APIError/TransportError/etc.
type PaymentError struct {
	InvoiceID     string
	Status        InvoiceStatus
	ErrCode       string
	FailureReason string

	// Metas are best-effort lookup results from PaymentErrorCatalog by ErrCode.
	// Some codes are duplicated in the docs, so we keep a slice.
	Metas []PaymentErrorMeta
}

func (e *PaymentError) Error() string {
	if e == nil {
		return ErrPaymentError.Error()
	}

	parts := []string{ErrPaymentError.Error()}

	if strings.TrimSpace(e.InvoiceID) != "" {
		parts = append(parts, "invoiceId="+strings.TrimSpace(e.InvoiceID))
	}
	if strings.TrimSpace(string(e.Status)) != "" {
		parts = append(parts, "status="+strings.TrimSpace(string(e.Status)))
	}
	if strings.TrimSpace(e.ErrCode) != "" {
		parts = append(parts, "errCode="+strings.TrimSpace(e.ErrCode))
	}
	if strings.TrimSpace(e.FailureReason) != "" {
		parts = append(parts, "reason="+strings.TrimSpace(e.FailureReason))
	}

	// If we have at least one meta, add one-line hint (without duplicating too much).
	if len(e.Metas) == 1 {
		if strings.TrimSpace(e.Metas[0].Contact) != "" {
			parts = append(parts, "contact="+strings.TrimSpace(e.Metas[0].Contact))
		}
	} else if len(e.Metas) > 1 {
		parts = append(parts, fmt.Sprintf("contact=%d-options", len(e.Metas)))
	}

	return strings.Join(parts, " ")
}

func (e *PaymentError) Is(target error) bool {
	return target == ErrPaymentError
}

// PrimaryMeta returns first matched metadata entry (if any).
func (e *PaymentError) PrimaryMeta() (*PaymentErrorMeta, bool) {
	if e == nil || len(e.Metas) == 0 {
		return nil, false
	}
	return &e.Metas[0], true
}

// Explanations returns deduplicated human-readable explanations.
func (e *PaymentError) Explanations() []string {
	if e == nil || len(e.Metas) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(e.Metas))
	out := make([]string, 0, len(e.Metas))
	for _, meta := range e.Metas {
		text := strings.TrimSpace(meta.Text)
		if text == "" {
			continue
		}
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		out = append(out, text)
	}
	return out
}

// Contacts returns deduplicated contact targets.
func (e *PaymentError) Contacts() []string {
	if e == nil || len(e.Metas) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(e.Metas))
	out := make([]string, 0, len(e.Metas))
	for _, meta := range e.Metas {
		contact := strings.TrimSpace(meta.Contact)
		if contact == "" {
			continue
		}
		if _, ok := seen[contact]; ok {
			continue
		}
		seen[contact] = struct{}{}
		out = append(out, contact)
	}
	sort.Strings(out)
	return out
}

// HandlingHints returns deduplicated next-step hints for operational handling.
func (e *PaymentError) HandlingHints() []string {
	if e == nil || len(e.Metas) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(e.Metas))
	out := make([]string, 0, len(e.Metas))
	for _, meta := range e.Metas {
		hint := strings.TrimSpace(meta.HandlingHint())
		if hint == "" {
			continue
		}
		if _, ok := seen[hint]; ok {
			continue
		}
		seen[hint] = struct{}{}
		out = append(out, hint)
	}
	return out
}

// PaymentErrorCatalog maps errCode -> one or more possible meta descriptions.
// Source: https://monobank.ua/api-docs/acquiring/dev/errors/payment
var PaymentErrorCatalog = map[string][]PaymentErrorMeta{
	"6":    {{Code: "6", Text: "Transaction is blocked by the issuing bank.", Contact: PaymentErrorContactIssuingBank}},
	"40":   {{Code: "40", Text: "Card is reported as lost. Spending is restricted.", Contact: PaymentErrorContactIssuingBank}},
	"41":   {{Code: "41", Text: "Card is reported as lost. Spending is restricted.", Contact: PaymentErrorContactIssuingBank}},
	"50":   {{Code: "50", Text: "Card spending is restricted.", Contact: PaymentErrorContactIssuingBank}},
	"51":   {{Code: "51", Text: "The card has expired.", Contact: PaymentErrorContactIssuingBank}},
	"52":   {{Code: "52", Text: "Card number is invalid.", Contact: PaymentErrorContactIssuingBank}},
	"54":   {{Code: "54", Text: "A technical failure occurred.", Contact: PaymentErrorContactIssuingBank}},
	"55":   {{Code: "55", Text: "Merchant configuration error.", Contact: PaymentErrorContactMonobank}},
	"56":   {{Code: "56", Text: "Card type does not support this payment.", Contact: PaymentErrorContactIssuingBank}},
	"57":   {{Code: "57", Text: "Transaction is not supported.", Contact: PaymentErrorContactIssuingBank}},
	"58":   {{Code: "58", Text: "Card spending for purchases is restricted.", Contact: PaymentErrorContactIssuingBank}, {Code: "58", Text: "Card spending is restricted.", Contact: PaymentErrorContactIssuingBank}},
	"59":   {{Code: "59", Text: "Insufficient funds to complete the purchase.", Contact: PaymentErrorContactIssuingBank}},
	"60":   {{Code: "60", Text: "Card spending transactions count limit exceeded.", Contact: PaymentErrorContactIssuingBank}},
	"61":   {{Code: "61", Text: "Card internet payment limit exceeded.", Contact: PaymentErrorContactIssuingBank}},
	"62":   {{Code: "62", Text: "PIN retry attempts limit is reached or exceeded.", Contact: PaymentErrorContactIssuingBank}},
	"63":   {{Code: "63", Text: "Card internet payment limit exceeded.", Contact: PaymentErrorContactIssuingBank}},
	"67":   {{Code: "67", Text: "Merchant configuration error.", Contact: PaymentErrorContactMonobank}},
	"68":   {{Code: "68", Text: "Payment system declined the transaction.", Contact: PaymentErrorContactIssuingBank}},
	"71":   {{Code: "71", Text: "Transaction is blocked by the issuing bank.", Contact: PaymentErrorContactIssuingBank}},
	"72":   {{Code: "72", Text: "Transaction is blocked by the issuing bank.", Contact: PaymentErrorContactIssuingBank}},
	"73":   {{Code: "73", Text: "Routing error.", Contact: PaymentErrorContactMonobank}},
	"74":   {{Code: "74", Text: "Merchant configuration error.", Contact: PaymentErrorContactMonobank}},
	"75":   {{Code: "75", Text: "Transaction is blocked by the issuing bank.", Contact: PaymentErrorContactIssuingBank}},
	"80":   {{Code: "80", Text: "Invalid CVV code.", Contact: PaymentErrorContactIssuingBank}},
	"81":   {{Code: "81", Text: "Invalid CVV2 code.", Contact: PaymentErrorContactIssuingBank}},
	"82":   {{Code: "82", Text: "Transaction is not allowed under these conditions.", Contact: PaymentErrorContactIssuingBank}, {Code: "82", Text: "Merchant configuration error.", Contact: PaymentErrorContactMonobank}},
	"83":   {{Code: "83", Text: "Card payment attempts limit exceeded.", Contact: PaymentErrorContactIssuingBank}},
	"84":   {{Code: "84", Text: "Invalid 3-D Secure CAVV value.", Contact: PaymentErrorContactMonobank}},
	"98":   {{Code: "98", Text: "Merchant configuration error.", Contact: PaymentErrorContactMonobank}},
	"1000": {{Code: "1000", Text: "Internal technical failure.", Contact: PaymentErrorContactMonobank}},
	"1005": {{Code: "1005", Text: "Internal technical failure.", Contact: PaymentErrorContactMonobank}},
	"1010": {{Code: "1010", Text: "Internal technical failure.", Contact: PaymentErrorContactMonobank}},
	"1014": {{Code: "1014", Text: "Full card details are required to process payment.", Contact: PaymentErrorContactCustomer}},
	"1034": {{Code: "1034", Text: "3-D Secure verification failed.", Contact: PaymentErrorContactIssuingBank}},
	"1035": {{Code: "1035", Text: "3-D Secure verification failed.", Contact: PaymentErrorContactIssuingBank}},
	"1036": {{Code: "1036", Text: "Internal technical failure.", Contact: PaymentErrorContactMonobank}},
	"1044": {{Code: "1044", Text: "Merchant configuration error.", Contact: PaymentErrorContactMonobank}},
	"1045": {{Code: "1045", Text: "3-D Secure verification failed.", Contact: PaymentErrorContactIssuingBank}},
	"1053": {{Code: "1053", Text: "Merchant configuration error.", Contact: PaymentErrorContactMonobank}},
	"1054": {{Code: "1054", Text: "3-D Secure verification failed.", Contact: PaymentErrorContactMonobank}},
	"1056": {{Code: "1056", Text: "Transfer is allowed only to cards issued by Ukrainian banks.", Contact: PaymentErrorContactMonobank}},
	"1064": {{Code: "1064", Text: "Payment is allowed only with Mastercard or Visa cards.", Contact: PaymentErrorContactIssuingBank}},
	"1066": {{Code: "1066", Text: "Merchant configuration error.", Contact: PaymentErrorContactMonobank}},
	"1077": {{Code: "1077", Text: "Payment amount is below minimum allowed amount (payment system settings).", Contact: PaymentErrorContactAPI}},
	"1080": {{Code: "1080", Text: "Card expiry date is invalid.", Contact: PaymentErrorContactIssuingBank}},
	"1090": {{Code: "1090", Text: "Customer information not found.", Contact: PaymentErrorContactMonobank}},
	"1115": {{Code: "1115", Text: "Merchant configuration error.", Contact: PaymentErrorContactMonobank}},
	"1121": {{Code: "1121", Text: "Merchant configuration error.", Contact: PaymentErrorContactMonobank}},
	"1145": {{Code: "1145", Text: "Minimum transfer amount is not met.", Contact: PaymentErrorContactMonobank}},
	"1165": {{Code: "1165", Text: "Transaction is blocked by the issuing bank.", Contact: PaymentErrorContactIssuingBank}},
	"1187": {{Code: "1187", Text: "Receiver name must be provided.", Contact: PaymentErrorContactAPI}},
	"1193": {{Code: "1193", Text: "Transaction is blocked by the issuing bank.", Contact: PaymentErrorContactIssuingBank}},
	"1194": {{Code: "1194", Text: "This top-up method works only with cards issued by other banks.", Contact: PaymentErrorContactMonobank}},
	"1200": {{Code: "1200", Text: "CVV code is required.", Contact: PaymentErrorContactIssuingBank}},
	"1405": {{Code: "1405", Text: "Payment system transfer limits reached.", Contact: PaymentErrorContactIssuingBank}},
	"1406": {{Code: "1406", Text: "Card is blocked by risk management.", Contact: PaymentErrorContactIssuingBank}},
	"1407": {{Code: "1407", Text: "Transaction is blocked by risk management.", Contact: PaymentErrorContactMonobank}},
	"1408": {{Code: "1408", Text: "Transaction is blocked by the issuing bank.", Contact: PaymentErrorContactIssuingBank}},
	"1411": {{Code: "1411", Text: "This type of operation from UAH cards is temporarily restricted.", Contact: PaymentErrorContactMonobank}},
	"1413": {{Code: "1413", Text: "Transaction is blocked by the issuing bank.", Contact: PaymentErrorContactIssuingBank}},
	"1419": {{Code: "1419", Text: "Card expiry date is invalid.", Contact: PaymentErrorContactIssuingBank}},
	"1420": {{Code: "1420", Text: "Internal technical failure.", Contact: PaymentErrorContactMonobank}},
	"1421": {{Code: "1421", Text: "3-D Secure verification failed.", Contact: PaymentErrorContactIssuingBank}},
	"1422": {{Code: "1422", Text: "Error occurred during 3-D Secure step.", Contact: PaymentErrorContactIssuingBank}},
	"1425": {{Code: "1425", Text: "Error occurred during 3-D Secure step.", Contact: PaymentErrorContactIssuingBank}},
	"1428": {{Code: "1428", Text: "Transaction is blocked by the issuing bank.", Contact: PaymentErrorContactIssuingBank}},
	"1429": {{Code: "1429", Text: "3-D Secure verification failed.", Contact: PaymentErrorContactIssuingBank}},
	"1433": {{Code: "1433", Text: "Check receiver first and last name. If data is invalid, bank can reject the transfer.", Contact: PaymentErrorContactMonobank}},
	"1436": {{Code: "1436", Text: "Payment rejected due to policy restrictions.", Contact: PaymentErrorContactMonobank}},
	"1439": {{Code: "1439", Text: "Operation is not allowed under the eRecovery program.", Contact: PaymentErrorContactMonobank}},
	"1458": {{Code: "1458", Text: "Transaction rejected at 3DS step.", Contact: PaymentErrorContactIssuingBank}},
	"8001": {{Code: "8001", Text: "Payment link has expired.", Contact: PaymentErrorContactCustomer}},
	"8002": {{Code: "8002", Text: "Customer cancelled the payment.", Contact: PaymentErrorContactCustomer}},
	"8003": {{Code: "8003", Text: "Technical failure occurred.", Contact: PaymentErrorContactMonobank}},
	"8004": {{Code: "8004", Text: "3-D Secure processing problem.", Contact: PaymentErrorContactIssuingBank}},
	"8005": {{Code: "8005", Text: "Payment acceptance limits exceeded.", Contact: PaymentErrorContactMonobank}},
	"8006": {{Code: "8006", Text: "Payment acceptance limits exceeded.", Contact: PaymentErrorContactMonobank}},
}

// LookupPaymentErrorMetas returns meta info for the given errCode.
func LookupPaymentErrorMetas(code string) ([]PaymentErrorMeta, bool) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, false
	}
	metas, ok := PaymentErrorCatalog[code]
	if !ok || len(metas) == 0 {
		return nil, false
	}
	// Return a copy to avoid accidental mutation from outside.
	out := make([]PaymentErrorMeta, len(metas))
	copy(out, metas)
	return out, true
}

// NewPaymentError builds a PaymentError from invoice status/webhook fields.
func NewPaymentError(invoiceID string, status InvoiceStatus, errCode string, failureReason string) *PaymentError {
	code := strings.TrimSpace(errCode)
	reason := strings.TrimSpace(failureReason)

	if code == "" && reason == "" {
		if !status.IsFailure() {
			return nil
		}
		reason = fmt.Sprintf("payment status indicates failure: %s", status)
	}

	pe := &PaymentError{
		InvoiceID:     strings.TrimSpace(invoiceID),
		Status:        status,
		ErrCode:       code,
		FailureReason: reason,
	}

	if metas, ok := LookupPaymentErrorMetas(code); ok {
		pe.Metas = metas
	}
	return pe
}
