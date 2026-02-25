package go_monobank

import (
	"encoding/base64"
	"testing"
)

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

func TestFiscalCheckConvenienceMethods(t *testing.T) {
	file := base64.StdEncoding.EncodeToString([]byte(`{"key":"value"}`))

	resp := &FiscalChecksResponse{
		Checks: []FiscalCheck{
			{ID: "1", Status: "done", TaxURL: "https://tax.example.com/check/1", File: file},
			{ID: "2", Status: "processing"},
			{ID: "3", Status: "failed"},
		},
	}

	if !resp.HasChecks() {
		t.Fatalf("expected HasChecks() to be true")
	}
	if got := len(resp.DoneChecks()); got != 1 {
		t.Fatalf("DoneChecks() len = %d, want 1", got)
	}
	if got := len(resp.PendingChecks()); got != 1 {
		t.Fatalf("PendingChecks() len = %d, want 1", got)
	}
	if got := len(resp.FailedChecks()); got != 1 {
		t.Fatalf("FailedChecks() len = %d, want 1", got)
	}

	first, ok := resp.FirstCheck()
	if !ok || first == nil || first.ID != "1" {
		t.Fatalf("unexpected FirstCheck() result: ok=%v first=%+v", ok, first)
	}
	last, ok := resp.LastCheck()
	if !ok || last == nil || last.ID != "3" {
		t.Fatalf("unexpected LastCheck() result: ok=%v last=%+v", ok, last)
	}

	parsedURL, err := first.ParsedTaxURL()
	if err != nil {
		t.Fatalf("ParsedTaxURL() unexpected error: %v", err)
	}
	if parsedURL == nil || parsedURL.String() != "https://tax.example.com/check/1" {
		t.Fatalf("unexpected ParsedTaxURL() value: %+v", parsedURL)
	}

	decoded, err := first.DecodedFile()
	if err != nil {
		t.Fatalf("DecodedFile() unexpected error: %v", err)
	}
	if string(decoded) != `{"key":"value"}` {
		t.Fatalf("DecodedFile() = %q, want %q", string(decoded), `{"key":"value"}`)
	}
}

func TestFiscalCheckConvenienceMethods_EdgeCases(t *testing.T) {
	var nilResp *FiscalChecksResponse
	if nilResp.HasChecks() {
		t.Fatalf("expected nil response HasChecks() to be false")
	}
	if first, ok := nilResp.FirstCheck(); ok || first != nil {
		t.Fatalf("expected nil response FirstCheck() to return nil,false")
	}
	if last, ok := nilResp.LastCheck(); ok || last != nil {
		t.Fatalf("expected nil response LastCheck() to return nil,false")
	}

	check := FiscalCheck{}
	if !check.IsPending() {
		t.Fatalf("empty status should be treated as pending")
	}

	check.TaxURL = "/relative"
	if _, err := check.ParsedTaxURL(); err == nil {
		t.Fatalf("expected ParsedTaxURL() error for relative URL")
	}

	check.File = "%%%bad-base64%%%"
	if _, err := check.DecodedFile(); err == nil {
		t.Fatalf("expected DecodedFile() error for invalid base64")
	}
}
