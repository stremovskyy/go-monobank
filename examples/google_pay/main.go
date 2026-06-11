package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	go_monobank "github.com/stremovskyy/go-monobank"
)

func main() {
	token := os.Getenv("MONO_TOKEN")
	if token == "" {
		log.Fatal("set MONO_TOKEN env")
	}
	payload := os.Getenv("GOOGLE_PAY_PAYLOAD")
	if payload == "" {
		log.Fatal("set GOOGLE_PAY_PAYLOAD env")
	}

	amount := int64(100)
	if s := os.Getenv("AMOUNT"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			log.Fatalf("invalid AMOUNT %q: %v", s, err)
		}
		amount = v
	}

	client := go_monobank.NewDefaultClient()
	req := go_monobank.NewRequest().
		WithToken(token).
		WithGoogleToken(payload).
		WithAmount(amount).
		WithCurrency(go_monobank.CurrencyUAH).
		WithInitiationKind(go_monobank.InitiationClient).
		WithReference("google-pay-001").
		WithDestination("Google Pay payment")

	if webhookURL := strings.TrimSpace(os.Getenv("WEBHOOK_URL")); webhookURL != "" {
		req.WithWebhookURL(webhookURL)
	}
	if redirectURL := strings.TrimSpace(os.Getenv("REDIRECT_URL")); redirectURL != "" {
		req.WithRedirectURL(redirectURL)
	}

	var (
		resp *go_monobank.WalletPaymentResponse
		err  error
	)
	if strings.EqualFold(os.Getenv("MONO_PAYMENT_TYPE"), string(go_monobank.PaymentTypeHold)) {
		resp, err = client.Hold(req)
	} else {
		req.WithPaymentType(go_monobank.PaymentTypeDebit)
		resp, err = client.Payment(req)
	}
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("invoiceId:", resp.InvoiceID)
	fmt.Println("status:", resp.Status)
	if resp.TDSURL != nil {
		fmt.Println("tdsUrl:", *resp.TDSURL)
	}
	if resp.FailureReason != nil {
		fmt.Println("failureReason:", *resp.FailureReason)
	}
}
