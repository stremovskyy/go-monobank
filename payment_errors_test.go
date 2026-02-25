package go_monobank

import (
	"errors"
	"strings"
	"testing"
)

func TestPaymentErrorCatalog_CoversMonobankDocsCodes(t *testing.T) {
	docCodes := []string{
		"6", "40", "41", "50", "51", "52", "54", "55", "56", "57", "58", "59", "60", "61", "62", "63",
		"67", "68", "71", "72", "73", "74", "75", "80", "81", "82", "83", "84", "98",
		"1000", "1005", "1010", "1014", "1034", "1035", "1036", "1044", "1045", "1053", "1054", "1056", "1064", "1066", "1077", "1080", "1090", "1115", "1121", "1145", "1165", "1187", "1193", "1194", "1200", "1405", "1406", "1407", "1408", "1411", "1413", "1419", "1420", "1421", "1422", "1425", "1428", "1429", "1433", "1436", "1439", "1458", "8001", "8002", "8003", "8004", "8005", "8006",
	}

	for _, code := range docCodes {
		metas, ok := LookupPaymentErrorMetas(code)
		if !ok || len(metas) == 0 {
			t.Fatalf("missing errCode in catalog: %s", code)
		}
		for _, m := range metas {
			if strings.TrimSpace(m.Text) == "" {
				t.Fatalf("empty text for code %s", code)
			}
			if strings.TrimSpace(m.Contact) == "" {
				t.Fatalf("empty contact for code %s", code)
			}
		}
	}

	if got := len(PaymentErrorCatalog["58"]); got != 2 {
		t.Fatalf("code 58 variants = %d, want 2", got)
	}
	if got := len(PaymentErrorCatalog["82"]); got != 2 {
		t.Fatalf("code 82 variants = %d, want 2", got)
	}
}

func TestPaymentErrorHandlingHints(t *testing.T) {
	tests := []struct {
		name    string
		contact string
		want    string
	}{
		{name: "issuer", contact: PaymentErrorContactIssuingBank, want: "issuing bank"},
		{name: "monobank", contact: PaymentErrorContactMonobank, want: "monobank support"},
		{name: "customer", contact: PaymentErrorContactCustomer, want: "customer"},
		{name: "api", contact: PaymentErrorContactAPI, want: "integration"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			h := (PaymentErrorMeta{Contact: tc.contact}).HandlingHint()
			if !strings.Contains(strings.ToLower(h), strings.ToLower(tc.want)) {
				t.Fatalf("unexpected handling hint %q, expected fragment %q", h, tc.want)
			}
		})
	}
}

func TestPaymentError_ConvenienceMethods(t *testing.T) {
	pe := NewPaymentError("inv-1", InvoiceFailure, "82", "declined")
	if pe == nil {
		t.Fatalf("expected payment error")
	}
	if !errors.Is(pe, ErrPaymentError) {
		t.Fatalf("errors.Is should match ErrPaymentError")
	}

	meta, ok := pe.PrimaryMeta()
	if !ok || meta == nil {
		t.Fatalf("expected primary meta")
	}
	if meta.Code != "82" {
		t.Fatalf("primary meta code = %s, want 82", meta.Code)
	}

	if got := pe.Explanations(); len(got) == 0 {
		t.Fatalf("expected explanations")
	}
	if got := pe.Contacts(); len(got) == 0 {
		t.Fatalf("expected contacts")
	}
	if got := pe.HandlingHints(); len(got) == 0 {
		t.Fatalf("expected handling hints")
	}
}

func TestNewPaymentError_StatusFailureWithoutDetails(t *testing.T) {
	pe := NewPaymentError("inv-2", InvoiceFailure, "", "")
	if pe == nil {
		t.Fatalf("expected non-nil payment error for failure status")
	}
	if !strings.Contains(pe.FailureReason, "failure") {
		t.Fatalf("unexpected fallback reason: %q", pe.FailureReason)
	}

	if got := NewPaymentError("inv-3", InvoiceSuccess, "", ""); got != nil {
		t.Fatalf("expected nil payment error for success without details")
	}
}

func TestRequireNoPaymentError(t *testing.T) {
	statusResp := &InvoiceStatusResponse{InvoiceID: "inv-s", Status: InvoiceFailure}
	if err := statusResp.RequireNoPaymentError(); err == nil {
		t.Fatalf("expected status RequireNoPaymentError to fail")
	}

	walletResp := &WalletPaymentResponse{InvoiceID: "inv-w", Status: InvoiceSuccess}
	if err := walletResp.RequireNoPaymentError(); err != nil {
		t.Fatalf("expected wallet RequireNoPaymentError to pass, got %v", err)
	}
}
