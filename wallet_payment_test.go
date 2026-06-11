package go_monobank

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stremovskyy/go-monobank/consts"
)

func TestApplePayMethodNormalizesFullPaymentPayloadToAToken(t *testing.T) {
	t.Parallel()

	tokenJSON := `{"paymentData":{"version":"EC_v1","data":"encrypted","signature":"signature","header":{"transactionId":"transaction-id","ephemeralPublicKey":"ephemeral-public-key","publicKeyHash":"public-key-hash"}},"paymentMethod":{"network":"Visa","type":"debit","displayName":"Visa 4242"},"transactionIdentifier":"transaction-id"}`
	fullPayment := `{"billingContact":{"countryCode":"UA"},"token":` + tokenJSON + `}`
	container := base64.StdEncoding.EncodeToString([]byte(fullPayment))

	method, err := NewApplePayMethod(container)
	if err != nil {
		t.Fatalf("NewApplePayMethod() error: %v", err)
	}

	request := NewRequest()
	request.PaymentMethod = method

	got, err := request.GetAToken()
	if err != nil {
		t.Fatalf("GetAToken() error: %v", err)
	}
	if got != tokenJSON {
		t.Fatalf("aToken = %s, want %s", got, tokenJSON)
	}
	if !request.IsApplePay() {
		t.Fatalf("expected request to be Apple Pay")
	}
}

func TestGooglePayMethodNormalizesPaymentDataToAToken(t *testing.T) {
	t.Parallel()

	tokenJSON := `{"signature":"signature","protocolVersion":"ECv2","signedMessage":"message"}`
	paymentData := `{"apiVersion":2,"apiVersionMinor":0,"paymentMethodData":{"type":"CARD","tokenizationData":{"type":"PAYMENT_GATEWAY","token":` + strconvQuote(tokenJSON) + `}}}`
	container := base64.StdEncoding.EncodeToString([]byte(paymentData))

	method, err := NewGooglePayMethod(container)
	if err != nil {
		t.Fatalf("NewGooglePayMethod() error: %v", err)
	}

	request := NewRequest()
	request.PaymentMethod = method

	got, err := request.GetAToken()
	if err != nil {
		t.Fatalf("GetAToken() error: %v", err)
	}
	if got != tokenJSON {
		t.Fatalf("aToken = %s, want %s", got, tokenJSON)
	}
	if !request.IsGooglePay() {
		t.Fatalf("expected request to be Google Pay")
	}
}

func TestPaymentDryRunSendsATokenForWalletPayment(t *testing.T) {
	t.Parallel()

	tokenJSON := `{"paymentData":{"version":"EC_v1"},"paymentMethod":{"network":"Visa"},"transactionIdentifier":"transaction-id"}`
	request := NewRequest().
		WithAmount(1234).
		WithCurrency(CurrencyUAH).
		WithPaymentType(PaymentTypeDebit).
		WithInitiationKind(InitiationClient).
		WithAToken(tokenJSON)

	got := captureWalletPaymentPayload(t, request)

	if got["aToken"] != tokenJSON {
		t.Fatalf("aToken = %v, want %s", got["aToken"], tokenJSON)
	}
	if _, ok := got["cardToken"]; ok {
		t.Fatalf("cardToken must be omitted for wallet-token payment: %+v", got)
	}
}

func TestPaymentDryRunKeepsCardTokenPayload(t *testing.T) {
	t.Parallel()

	request := NewRequest().
		WithAmount(1234).
		WithCurrency(CurrencyUAH).
		WithPaymentType(PaymentTypeDebit).
		WithInitiationKind(InitiationMerchant).
		WithCardToken("card-token")

	got := captureWalletPaymentPayload(t, request)

	if got["cardToken"] != "card-token" {
		t.Fatalf("cardToken = %v, want card-token", got["cardToken"])
	}
	if _, ok := got["aToken"]; ok {
		t.Fatalf("aToken must be omitted for card-token payment: %+v", got)
	}
}

func TestPaymentRejectsMultiplePaymentSources(t *testing.T) {
	t.Parallel()

	client := NewClient(WithToken("merchant-token"))
	request := NewRequest().
		WithAmount(1234).
		WithCurrency(CurrencyUAH).
		WithInitiationKind(InitiationMerchant).
		WithCardToken("card-token").
		WithAToken(`{"paymentData":{"version":"EC_v1"}}`)

	_, err := client.Payment(request, DryRun())
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestHoldUsesWalletPaymentWithHoldPaymentType(t *testing.T) {
	t.Parallel()

	tokenJSON := `{"paymentData":{"version":"EC_v1"},"paymentMethod":{"network":"Visa"},"transactionIdentifier":"transaction-id"}`
	request := NewRequest().
		WithAmount(1234).
		WithCurrency(CurrencyUAH).
		WithInitiationKind(InitiationClient).
		WithAToken(tokenJSON)

	var endpoint string
	var payload any
	client := NewClient(WithToken("merchant-token"))
	_, err := client.Hold(
		request,
		DryRun(func(gotEndpoint string, gotPayload any) {
			endpoint = gotEndpoint
			payload = gotPayload
		}),
	)
	if err != nil {
		t.Fatalf("Hold() unexpected error: %v", err)
	}
	if !strings.HasSuffix(endpoint, consts.PathWalletPayment) {
		t.Fatalf("endpoint = %q, want suffix %q", endpoint, consts.PathWalletPayment)
	}

	got := decodePayloadMap(t, payload)
	if got["aToken"] != tokenJSON {
		t.Fatalf("aToken = %v, want %s", got["aToken"], tokenJSON)
	}
	if got["paymentType"] != string(PaymentTypeHold) {
		t.Fatalf("paymentType = %v, want %s", got["paymentType"], PaymentTypeHold)
	}
}

func TestNewWalletPaymentMethodRejectsInvalidApplePayload(t *testing.T) {
	t.Parallel()

	if _, err := NewApplePayMethod("not-json"); err == nil {
		t.Fatalf("expected invalid Apple Pay payload error")
	}
}

func captureWalletPaymentPayload(t *testing.T, request *Request) map[string]any {
	t.Helper()

	var endpoint string
	var payload any
	client := NewClient(WithToken("merchant-token"))

	_, err := client.Payment(
		request,
		DryRun(func(gotEndpoint string, gotPayload any) {
			endpoint = gotEndpoint
			payload = gotPayload
		}),
	)
	if err != nil {
		t.Fatalf("Payment() unexpected error: %v", err)
	}
	if !strings.HasSuffix(endpoint, consts.PathWalletPayment) {
		t.Fatalf("endpoint = %q, want suffix %q", endpoint, consts.PathWalletPayment)
	}

	return decodePayloadMap(t, payload)
}

func decodePayloadMap(t *testing.T, payload any) map[string]any {
	t.Helper()

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payloadJSON, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	return got
}

func strconvQuote(value string) string {
	quoted, _ := json.Marshal(value)
	return string(quoted)
}
