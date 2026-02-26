package go_monobank

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stremovskyy/recorder"
)

type captureStorage struct {
	mu      sync.Mutex
	records []recorder.Record
}

func (s *captureStorage) Save(_ context.Context, record recorder.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, record)
	return nil
}

func (s *captureStorage) Load(_ context.Context, _ recorder.RecordType, _ string) ([]byte, error) {
	return nil, nil
}

func (s *captureStorage) FindByTag(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (s *captureStorage) snapshot() []recorder.Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]recorder.Record, len(s.records))
	copy(out, s.records)
	return out
}

type errorRoundTripper struct {
	err error
}

func (e errorRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, e.err
}

func TestStatusRecordsRequestAndResponse(t *testing.T) {
	storage := &captureStorage{}
	rec := recorder.New(storage)

	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/merchant/invoice/status" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"invoiceId":"inv-1","status":"success","amount":100,"ccy":980,"createdDate":"2026-02-26T10:00:00Z","modifiedDate":"2026-02-26T10:00:00Z"}`))
			},
		),
	)
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithRecorder(rec),
		WithToken("merchant-token"),
	)

	request := NewRequest().WithInvoiceID("inv-1")
	_, err := client.Status(request)
	if err != nil {
		t.Fatalf("status request failed: %v", err)
	}

	records := storage.snapshot()
	if len(records) != 2 {
		t.Fatalf("expected request+response records, got %d", len(records))
	}

	if records[0].Type != recorder.RecordTypeRequest {
		t.Fatalf("first record must be request, got %s", records[0].Type)
	}
	if records[1].Type != recorder.RecordTypeResponse {
		t.Fatalf("second record must be response, got %s", records[1].Type)
	}

	if records[0].RequestID == "" || records[0].RequestID != records[1].RequestID {
		t.Fatalf("request and response must have same non-empty request id")
	}

	if records[0].Tags["invoice_id"] != "inv-1" {
		t.Fatalf("expected invoice_id tag, got %q", records[0].Tags["invoice_id"])
	}
	if records[0].Tags["operation"] != "status" {
		t.Fatalf("expected operation=status, got %q", records[0].Tags["operation"])
	}
	if records[1].Tags["status_code"] != "200" {
		t.Fatalf("expected status_code=200, got %q", records[1].Tags["status_code"])
	}

	if len(records[0].Payload) == 0 {
		t.Fatalf("request payload should not be empty")
	}
}

func TestPaymentRecordsErrorOnTransportFailure(t *testing.T) {
	storage := &captureStorage{}
	rec := recorder.New(storage)

	httpClient := &http.Client{
		Transport: errorRoundTripper{err: errors.New("network down")},
	}

	client := NewClient(
		WithBaseURL("https://api.monobank.ua"),
		WithRecorder(rec),
		WithClient(httpClient),
		WithToken("merchant-token"),
	)

	request := NewRequest().
		WithCardToken("card-token").
		WithAmount(100).
		WithCurrency(CurrencyUAH).
		WithInitiationKind(InitiationMerchant).
		WithReference("order-1")

	_, err := client.Payment(request)
	if err == nil {
		t.Fatalf("expected payment error")
	}

	records := storage.snapshot()
	if len(records) < 2 {
		t.Fatalf("expected at least request and error records, got %d", len(records))
	}

	hasRequest := false
	hasError := false
	for _, record := range records {
		if record.Type == recorder.RecordTypeRequest {
			hasRequest = true
			if record.Tags["invoice_id"] != "order-1" {
				t.Fatalf("expected invoice_id=order-1 in request tags, got %q", record.Tags["invoice_id"])
			}
			if record.Tags["operation"] != "payment" {
				t.Fatalf("expected operation=payment in request tags, got %q", record.Tags["operation"])
			}
		}
		if record.Type == recorder.RecordTypeError {
			hasError = true
			if !strings.Contains(string(record.Payload), "network down") {
				t.Fatalf("unexpected error payload: %s", string(record.Payload))
			}
			if record.Tags["invoice_id"] != "order-1" {
				t.Fatalf("expected invoice_id=order-1 in error tags, got %q", record.Tags["invoice_id"])
			}
			if record.Tags["operation"] != "payment" {
				t.Fatalf("expected operation=payment in error tags, got %q", record.Tags["operation"])
			}
		}
	}

	if !hasRequest {
		t.Fatalf("request record is missing")
	}
	if !hasError {
		t.Fatalf("error record is missing")
	}
}
