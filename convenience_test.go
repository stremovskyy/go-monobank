package go_monobank

import "testing"

func TestRequestConvenienceMethods(t *testing.T) {
	req := NewRequest().
		WithWebhookURL("  https://example.com/webhook  ").
		WithWalletID("wallet-1").
		EnableSaveCard()

	if req.GetWebHookURL() == nil || *req.GetWebHookURL() != "https://example.com/webhook" {
		t.Fatalf("unexpected webhook url: %+v", req.GetWebHookURL())
	}
	if got := req.GetWalletID(); got != "wallet-1" {
		t.Fatalf("wallet id mismatch: got %q want %q", got, "wallet-1")
	}
	if !req.ShouldSaveCard() {
		t.Fatalf("expected save card to be enabled")
	}

	req.DisableSaveCard()
	if req.ShouldSaveCard() {
		t.Fatalf("expected save card to be disabled")
	}

	req.SaveCard("wallet-2")
	if got := req.GetWalletID(); got != "wallet-2" {
		t.Fatalf("wallet id mismatch after SaveCard: got %q want %q", got, "wallet-2")
	}
	if !req.ShouldSaveCard() {
		t.Fatalf("expected save card to be enabled after SaveCard")
	}
}

func TestInvoiceStatusConvenienceMethods(t *testing.T) {
	tests := []struct {
		name    string
		status  InvoiceStatus
		final   bool
		success bool
		failure bool
		pending bool
	}{
		{name: "created", status: InvoiceCreated, pending: true},
		{name: "processing", status: InvoiceProcessing, pending: true},
		{name: "success", status: InvoiceSuccess, final: true, success: true},
		{name: "failure", status: InvoiceFailure, final: true, failure: true},
		{name: "reversed", status: InvoiceReversed, final: true, failure: true},
		{name: "expired", status: InvoiceExpired, final: true, failure: true},
		{name: "unknown", status: InvoiceStatus("unknown")},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(
			tc.name, func(t *testing.T) {
				if got := tc.status.IsFinal(); got != tc.final {
					t.Fatalf("IsFinal() = %v, want %v", got, tc.final)
				}
				if got := tc.status.IsSuccess(); got != tc.success {
					t.Fatalf("IsSuccess() = %v, want %v", got, tc.success)
				}
				if got := tc.status.IsFailure(); got != tc.failure {
					t.Fatalf("IsFailure() = %v, want %v", got, tc.failure)
				}
				if got := tc.status.IsPending(); got != tc.pending {
					t.Fatalf("IsPending() = %v, want %v", got, tc.pending)
				}
			},
		)
	}
}

func TestResponseConvenienceMethods(t *testing.T) {
	statusResp := &InvoiceStatusResponse{Status: InvoiceSuccess}
	if !statusResp.IsSuccess() || !statusResp.IsFinal() || statusResp.IsFailure() || statusResp.IsPending() {
		t.Fatalf("unexpected InvoiceStatusResponse helper results for success")
	}

	tdsURL := "https://acs.example.com/challenge"
	payResp := &WalletPaymentResponse{
		Status: InvoiceProcessing,
		TDSURL: &tdsURL,
	}
	if !payResp.Requires3DS() {
		t.Fatalf("expected Requires3DS to be true")
	}
	if payResp.IsFinal() || !payResp.IsPending() || payResp.IsFailure() || payResp.IsSuccess() {
		t.Fatalf("unexpected WalletPaymentResponse helper results for processing")
	}

	blankURL := "   "
	payResp.TDSURL = &blankURL
	if payResp.Requires3DS() {
		t.Fatalf("expected Requires3DS to be false for blank tdsUrl")
	}
}

func TestInvoiceCreateResponseParsedPageURL(t *testing.T) {
	var nilResp *InvoiceCreateResponse
	if _, err := nilResp.ParsedPageURL(); err == nil {
		t.Fatalf("expected error for nil receiver")
	}

	resp := &InvoiceCreateResponse{
		PageURL: "https://pay.monobank.ua/example",
	}
	parsed, err := resp.ParsedPageURL()
	if err != nil {
		t.Fatalf("ParsedPageURL() unexpected error: %v", err)
	}
	if got := parsed.String(); got != "https://pay.monobank.ua/example" {
		t.Fatalf("ParsedPageURL() = %q, want %q", got, "https://pay.monobank.ua/example")
	}

	resp.PageURL = "/relative"
	if _, err := resp.ParsedPageURL(); err == nil {
		t.Fatalf("expected error for relative URL")
	}
}
