package go_monobank

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWalletRequiresToken(t *testing.T) {
	t.Parallel()

	client := NewDefaultClient()
	request := NewRequest().WithWalletID("wallet-123")

	_, err := client.Wallet(request, DryRun())
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestWalletRequiresWalletID(t *testing.T) {
	t.Parallel()

	client := NewClient(WithToken("merchant-token"))
	request := NewRequest()

	_, err := client.Wallet(request, DryRun())
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestWalletDryRunCapturesEndpointAndWalletID(t *testing.T) {
	t.Parallel()

	client := NewClient(WithToken("merchant-token"))
	request := NewRequest().WithWalletID("wallet 123")

	var endpoint string
	var payload any

	_, err := client.Wallet(
		request,
		DryRun(
			func(gotEndpoint string, gotPayload any) {
				endpoint = gotEndpoint
				payload = gotPayload
			},
		),
	)
	if err != nil {
		t.Fatalf("Wallet() unexpected error: %v", err)
	}

	if !strings.HasSuffix(endpoint, "/api/merchant/wallet?walletId=wallet+123") {
		t.Fatalf("unexpected dry-run endpoint: %s", endpoint)
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal dry-run payload: %v", err)
	}

	var got map[string]string
	if err = json.Unmarshal(payloadJSON, &got); err != nil {
		t.Fatalf("unmarshal dry-run payload: %v", err)
	}

	if got["walletId"] != "wallet 123" {
		t.Fatalf("unexpected walletId payload: got %q want wallet 123", got["walletId"])
	}
}

func TestWalletDecodesMaskedPan(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("unexpected method: %s", r.Method)
				}

				if r.URL.Path != "/api/merchant/wallet" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}

				if r.URL.Query().Get("walletId") != "wallet-123" {
					t.Fatalf("unexpected walletId: %s", r.URL.RawQuery)
				}

				if r.Header.Get("X-Token") != "merchant-token" {
					t.Fatalf("unexpected token header: %q", r.Header.Get("X-Token"))
				}

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(
					[]byte(`{"wallet":[{"cardToken":"tok_1","maskedPan":"424242******4242","country":"804"}]}`),
				)
			},
		),
	)
	defer server.Close()

	client := NewClient(WithToken("merchant-token"), WithBaseURL(server.URL))
	request := NewRequest().WithWalletID("wallet-123")

	response, err := client.Wallet(request)
	if err != nil {
		t.Fatalf("Wallet() unexpected error: %v", err)
	}

	if response == nil || len(response.Wallet) != 1 {
		t.Fatalf("expected one wallet card, got %#v", response)
	}

	card := response.Wallet[0]
	if card.CardToken != "tok_1" {
		t.Fatalf("unexpected card token: %q", card.CardToken)
	}
	if card.MaskedPan != "424242******4242" {
		t.Fatalf("unexpected masked pan: %q", card.MaskedPan)
	}
	if card.Country != "804" {
		t.Fatalf("unexpected country: %q", card.Country)
	}
}
