package go_monobank

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestVerificationAllowsZeroAmountForVerificationPaymentType(t *testing.T) {
	t.Parallel()

	client := NewClient(WithToken("merchant-token"))
	request := NewRequest().
		WithAmount(0).
		WithCurrency(CurrencyUAH).
		WithPaymentType(PaymentTypeVerification).
		WithReference("order-123").
		WithDestination("Card verification").
		SaveCard("wallet-123")

	var endpoint string
	var payload any

	_, err := client.Verification(
		request,
		DryRun(
			func(gotEndpoint string, gotPayload any) {
				endpoint = gotEndpoint
				payload = gotPayload
			},
		),
	)
	if err != nil {
		t.Fatalf("Verification() unexpected error: %v", err)
	}
	if endpoint == "" {
		t.Fatalf("expected dry-run endpoint to be captured")
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal dry-run payload: %v", err)
	}

	var got struct {
		Amount       int64        `json:"amount"`
		Currency     CurrencyCode `json:"ccy"`
		PaymentType  PaymentType  `json:"paymentType"`
		SaveCardData struct {
			SaveCard bool   `json:"saveCard"`
			WalletID string `json:"walletId"`
		} `json:"saveCardData"`
	}
	if err = json.Unmarshal(payloadJSON, &got); err != nil {
		t.Fatalf("unmarshal dry-run payload: %v", err)
	}

	if got.Amount != 0 {
		t.Fatalf("amount = %d, want 0", got.Amount)
	}
	if got.Currency != CurrencyUAH {
		t.Fatalf("ccy = %d, want %d", got.Currency, CurrencyUAH)
	}
	if got.PaymentType != PaymentTypeVerification {
		t.Fatalf("paymentType = %q, want %q", got.PaymentType, PaymentTypeVerification)
	}
	if !got.SaveCardData.SaveCard {
		t.Fatalf("expected saveCardData.saveCard=true")
	}
	if got.SaveCardData.WalletID != "wallet-123" {
		t.Fatalf("walletId = %q, want wallet-123", got.SaveCardData.WalletID)
	}
}

func TestVerificationRejectsPositiveAmountForVerificationPaymentType(t *testing.T) {
	t.Parallel()

	client := NewClient(WithToken("merchant-token"))
	request := NewRequest().
		WithAmount(100).
		WithCurrency(CurrencyUAH).
		WithPaymentType(PaymentTypeVerification).
		SaveCard("wallet-123")

	_, err := client.Verification(request, DryRun())
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestVerificationRejectsMissingSaveCardForVerificationPaymentType(t *testing.T) {
	t.Parallel()

	client := NewClient(WithToken("merchant-token"))
	request := NewRequest().
		WithAmount(0).
		WithCurrency(CurrencyUAH).
		WithPaymentType(PaymentTypeVerification)

	_, err := client.Verification(request, DryRun())
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestVerificationKeepsPositiveAmountForDebitAndHoldPaymentTypes(t *testing.T) {
	t.Parallel()

	for _, paymentType := range []PaymentType{PaymentTypeDebit, PaymentTypeHold} {
		paymentType := paymentType
		t.Run(
			string(paymentType), func(t *testing.T) {
				t.Parallel()

				client := NewClient(WithToken("merchant-token"))
				request := NewRequest().
					WithAmount(100).
					WithCurrency(CurrencyUAH).
					WithPaymentType(paymentType)

				if _, err := client.Verification(request, DryRun()); err != nil {
					t.Fatalf("Verification() unexpected error: %v", err)
				}
			},
		)
	}
}
