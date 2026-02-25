package main

import (
	"fmt"
	"log"
	"os"

	go_monobank "github.com/stremovskyy/go-monobank"
	log2 "github.com/stremovskyy/go-monobank/log"
)

func main() {
	token := os.Getenv("MONO_TOKEN")
	if token == "" {
		log.Fatal("set MONO_TOKEN env")
	}

	walletID := os.Getenv("WALLET_ID")
	if walletID == "" {
		walletID = "wallet-id-demo" // replace in real usage
	}

	redirectURL := os.Getenv("REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = "https://example.com/return"
	}

	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		webhookURL = "https://example.com/mono/webhook"
	}

	client := go_monobank.NewDefaultClient()

	client.SetLogLevel(log2.Debug)

	req := go_monobank.NewRequest().
		WithToken(token).
		WithAmount(100).
		WithCurrency(go_monobank.CurrencyUAH).
		WithRedirectURL(redirectURL).
		WithWebHookURL(webhookURL).
		WithReference("verify-card-001").
		WithDestination("Card verification").
		SaveCard(walletID)

	resp, err := client.Verification(req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Method - Verification - invoiceId:", resp.InvoiceID)
	fmt.Println("Method - Verification - pageUrl:", resp.PageURL)

	link, err := client.VerificationLink(req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Method - VerificationLink - parsedUrl:", link.String())

}
